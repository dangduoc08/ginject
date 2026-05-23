package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

type multipartFile struct {
	field    string
	filename string
	reader   io.Reader
}

type requestBuilder struct {
	client      *httpClient
	method      string
	path        string
	ctx         context.Context
	headers     map[string]string
	queryParams url.Values
	// body
	jsonBody    any
	rawBody     io.Reader
	bodyBytes   []byte
	contentType string
	// form / multipart
	formFields url.Values
	files      []multipartFile
	// per-request overrides
	retryCount   int
	retryInitial time.Duration
	retryMax     time.Duration
	timeout      time.Duration
	// streaming
	isStream   bool
	onProgress func(Progress)
}

func (rb *requestBuilder) Context(ctx context.Context) RequestBuilder {
	rb.ctx = ctx
	return rb
}

func (rb *requestBuilder) Header(key, value string) RequestBuilder {
	rb.headers[key] = value
	return rb
}

func (rb *requestBuilder) Headers(headers map[string]string) RequestBuilder {
	for k, v := range headers {
		rb.headers[k] = v
	}
	return rb
}

func (rb *requestBuilder) Query(key string, value any) RequestBuilder {
	switch v := value.(type) {
	case []string:
		for _, s := range v {
			rb.queryParams.Add(key, s)
		}
	case string:
		rb.queryParams.Set(key, v)
	default:
		rb.queryParams.Set(key, fmt.Sprint(v))
	}
	return rb
}

func (rb *requestBuilder) JSON(v any) RequestBuilder {
	rb.jsonBody = v
	return rb
}

func (rb *requestBuilder) Form(v any) RequestBuilder {
	switch data := v.(type) {
	case map[string]string:
		for k, val := range data {
			rb.formFields.Set(k, val)
		}
	case url.Values:
		for k, vv := range data {
			rb.formFields[k] = vv
		}
	case map[string]any:
		for k, val := range data {
			rb.formFields.Set(k, fmt.Sprint(val))
		}
	}
	return rb
}

func (rb *requestBuilder) Body(r io.Reader) RequestBuilder {
	rb.rawBody = r
	return rb
}

func (rb *requestBuilder) File(field, filename string, r io.Reader) RequestBuilder {
	rb.files = append(rb.files, multipartFile{field: field, filename: filename, reader: r})
	return rb
}

func (rb *requestBuilder) Field(key, value string) RequestBuilder {
	rb.formFields.Set(key, value)
	return rb
}

func (rb *requestBuilder) Timeout(d time.Duration) RequestBuilder {
	rb.timeout = d
	return rb
}

func (rb *requestBuilder) Retry(count int) RequestBuilder {
	rb.retryCount = count
	return rb
}

func (rb *requestBuilder) RetryBackoff(initial, max time.Duration) RequestBuilder {
	rb.retryInitial = initial
	rb.retryMax = max
	return rb
}

func (rb *requestBuilder) Stream() RequestBuilder {
	rb.isStream = true
	return rb
}

func (rb *requestBuilder) SSE() RequestBuilder {
	rb.isStream = true
	rb.headers["Accept"] = "text/event-stream"
	return rb
}

func (rb *requestBuilder) OnProgress(fn func(Progress)) RequestBuilder {
	rb.onProgress = fn
	return rb
}

func (rb *requestBuilder) resolveURL() (string, error) {
	path := rb.path

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if len(rb.queryParams) == 0 {
			return path, nil
		}
		u, err := url.Parse(path)
		if err != nil {
			return "", err
		}
		q := u.Query()
		for k, vv := range rb.queryParams {
			for _, v := range vv {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
		return u.String(), nil
	}

	rb.client.mu.RLock()
	base := rb.client.baseURL
	rb.client.mu.RUnlock()

	fullURL := base
	if !strings.HasPrefix(path, "/") {
		fullURL += "/"
	}
	fullURL += path

	if len(rb.queryParams) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return "", err
		}
		q := u.Query()
		for k, vv := range rb.queryParams {
			for _, v := range vv {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
		return u.String(), nil
	}

	return fullURL, nil
}

func (rb *requestBuilder) buildBody() (io.Reader, string, error) {
	if rb.jsonBody != nil {
		data, err := json.Marshal(rb.jsonBody)
		if err != nil {
			return nil, "", fmt.Errorf("httpclient: JSON marshal: %w", err)
		}
		rb.bodyBytes = data
		return bytes.NewReader(data), "application/json", nil
	}

	if len(rb.files) > 0 {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		for k, vv := range rb.formFields {
			for _, v := range vv {
				if err := mw.WriteField(k, v); err != nil {
					return nil, "", err
				}
			}
		}
		for _, f := range rb.files {
			fw, err := mw.CreateFormFile(f.field, f.filename)
			if err != nil {
				return nil, "", err
			}
			if _, err := io.Copy(fw, f.reader); err != nil {
				return nil, "", err
			}
		}
		if err := mw.Close(); err != nil {
			return nil, "", err
		}
		rb.bodyBytes = buf.Bytes()
		return bytes.NewReader(rb.bodyBytes), mw.FormDataContentType(), nil
	}

	if len(rb.formFields) > 0 {
		encoded := rb.formFields.Encode()
		rb.bodyBytes = []byte(encoded)
		return bytes.NewReader(rb.bodyBytes), "application/x-www-form-urlencoded", nil
	}

	if rb.rawBody != nil {
		return rb.rawBody, rb.contentType, nil
	}

	return nil, "", nil
}

func (rb *requestBuilder) buildRequest(ctx context.Context) (*http.Request, error) {
	rawURL, err := rb.resolveURL()
	if err != nil {
		return nil, err
	}

	body, contentType, err := rb.buildBody()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, rb.method, rawURL, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	rb.client.mu.RLock()
	for k, v := range rb.client.defaultHeaders {
		req.Header.Set(k, v)
	}
	rb.client.mu.RUnlock()

	for k, v := range rb.headers {
		req.Header.Set(k, v)
	}

	if rb.bodyBytes != nil {
		bb := rb.bodyBytes
		req.ContentLength = int64(len(bb))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bb)), nil
		}
	}

	return req, nil
}

func (rb *requestBuilder) Send() (*Response, error) {
	ctx := rb.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var cancel context.CancelFunc
	if rb.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, rb.timeout)
	}

	req, err := rb.buildRequest(ctx)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, &Error{Cause: err}
	}

	tc := newTimingCollector()
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), tc.clientTrace()))

	rb.client.mu.RLock()
	beforeHooks := make([]func(*http.Request) error, len(rb.client.beforeRequest))
	copy(beforeHooks, rb.client.beforeRequest)
	middlewares := make([]Middleware, len(rb.client.middlewares))
	copy(middlewares, rb.client.middlewares)
	debug := rb.client.debugMode
	afterHooks := make([]func(*Response) error, len(rb.client.afterResponse))
	copy(afterHooks, rb.client.afterResponse)
	errHooks := make([]func(error), len(rb.client.onError))
	copy(errHooks, rb.client.onError)
	validateStatus := rb.client.validateStatus
	retryCount := rb.client.retryCount
	retryInitial := rb.client.retryInitial
	retryMax := rb.client.retryMax
	rb.client.mu.RUnlock()

	for _, fn := range beforeHooks {
		if err := fn(req); err != nil {
			if cancel != nil {
				cancel()
			}
			return nil, &Error{Request: req, Cause: err}
		}
	}

	if debug {
		debugRequest(req)
	}

	// Per-request overrides take precedence.
	if rb.retryCount > 0 {
		retryCount = rb.retryCount
	}
	if rb.retryInitial > 0 {
		retryInitial = rb.retryInitial
	}
	if rb.retryMax > 0 {
		retryMax = rb.retryMax
	}

	finalHandler := rb.client.buildFinalHandler(tc, rb.isStream, rb.onProgress)
	handler := buildChain(middlewares, finalHandler)

	execFn := func() (*Response, error) {
		reqCopy := req.Clone(req.Context())
		if req.GetBody != nil {
			b, getErr := req.GetBody()
			if getErr != nil {
				return nil, getErr
			}
			reqCopy.Body = b
		}
		return handler(reqCopy)
	}

	var resp *Response
	if retryCount > 0 {
		rc := retryConfig{count: retryCount, initialWait: retryInitial, maxWait: retryMax}
		resp, err = rc.execute(execFn)
	} else {
		resp, err = execFn()
	}

	if err != nil {
		if cancel != nil {
			cancel()
		}
		for _, fn := range errHooks {
			fn(err)
		}
		return nil, err
	}

	if validateStatus != nil && !validateStatus(resp.StatusCode) {
		httpErr := &Error{Request: req, Response: resp,
			Cause: fmt.Errorf("status %d", resp.StatusCode)}
		if cancel != nil {
			cancel()
		}
		for _, fn := range errHooks {
			fn(httpErr)
		}
		return nil, httpErr
	}

	// For streaming responses, bind cancel to the body close so the context
	// is not cancelled before the caller finishes reading.
	if rb.isStream && cancel != nil {
		resp.BodyStream = &cancelReadCloser{ReadCloser: resp.BodyStream, cancel: cancel}
		cancel = nil
	}

	if cancel != nil {
		cancel()
	}

	for _, fn := range afterHooks {
		if err := fn(resp); err != nil {
			return nil, &Error{Request: req, Response: resp, Cause: err}
		}
	}

	if debug {
		debugResponse(resp, resp.Timing)
	}

	return resp, nil
}
