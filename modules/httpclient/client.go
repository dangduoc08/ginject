package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

type httpClient struct {
	mu              sync.RWMutex
	baseURL         string
	defaultHeaders  map[string]string
	middlewares     []Middleware
	beforeRequest   []func(*http.Request) error
	afterResponse   []func(*Response) error
	onError         []func(error)
	retryCount      int
	retryInitial    time.Duration
	retryMax        time.Duration
	validateStatus  func(int) bool
	validateHost    func(string) bool
	tlsConfig       *tls.Config
	requireHTTPS    bool
	maxResponseSize int64
	debugMode       bool
	hasCookieJar    bool
	underlying      *http.Client
}

func newHTTPClient(opts *HTTPClientModuleOptions) *httpClient {
	c := &httpClient{
		defaultHeaders:  make(map[string]string),
		validateStatus:  func(code int) bool { return code >= 200 && code < 400 },
		maxResponseSize: 32 << 20, // 32 MB
		retryInitial:    100 * time.Millisecond,
		retryMax:        5 * time.Second,
	}

	if opts != nil {
		if opts.BaseURL != "" {
			c.baseURL = strings.TrimRight(opts.BaseURL, "/")
		}
		for k, v := range opts.Headers {
			c.defaultHeaders[k] = v
		}
	}

	c.underlying = &http.Client{
		Transport: buildTransport(nil, false, nil),
	}
	if opts != nil && opts.Timeout > 0 {
		c.underlying.Timeout = opts.Timeout
	}

	return c
}

func (c *httpClient) rebuildTransport() {
	c.underlying.Transport = buildTransport(c.tlsConfig, c.requireHTTPS, c.validateHost)
}

func (c *httpClient) newRequest(method, path string) *requestBuilder {
	return &requestBuilder{
		client:      c,
		method:      method,
		path:        path,
		headers:     make(map[string]string),
		queryParams: url.Values{},
		formFields:  url.Values{},
	}
}

func (c *httpClient) Get(path string) RequestBuilder    { return c.newRequest(http.MethodGet, path) }
func (c *httpClient) Post(path string) RequestBuilder   { return c.newRequest(http.MethodPost, path) }
func (c *httpClient) Put(path string) RequestBuilder    { return c.newRequest(http.MethodPut, path) }
func (c *httpClient) Patch(path string) RequestBuilder  { return c.newRequest(http.MethodPatch, path) }
func (c *httpClient) Delete(path string) RequestBuilder { return c.newRequest(http.MethodDelete, path) }
func (c *httpClient) Head(path string) RequestBuilder   { return c.newRequest(http.MethodHead, path) }
func (c *httpClient) Options(path string) RequestBuilder {
	return c.newRequest(http.MethodOptions, path)
}

func (c *httpClient) Use(middlewares ...Middleware) {
	c.mu.Lock()
	c.middlewares = append(c.middlewares, middlewares...)
	c.mu.Unlock()
}

func (c *httpClient) SetBaseURL(u string) {
	c.mu.Lock()
	c.baseURL = strings.TrimRight(u, "/")
	c.mu.Unlock()
}

func (c *httpClient) SetHeader(key, value string) {
	c.mu.Lock()
	c.defaultHeaders[key] = value
	c.mu.Unlock()
}

func (c *httpClient) SetHeaders(headers map[string]string) {
	c.mu.Lock()
	for k, v := range headers {
		c.defaultHeaders[k] = v
	}
	c.mu.Unlock()
}

func (c *httpClient) SetTimeout(d time.Duration) {
	c.mu.Lock()
	c.underlying.Timeout = d
	c.mu.Unlock()
}

func (c *httpClient) SetRetry(count int) {
	c.mu.Lock()
	c.retryCount = count
	c.mu.Unlock()
}

func (c *httpClient) SetRetryBackoff(initial, max time.Duration) {
	c.mu.Lock()
	c.retryInitial = initial
	c.retryMax = max
	c.mu.Unlock()
}

func (c *httpClient) EnableDebug() {
	c.mu.Lock()
	c.debugMode = true
	c.mu.Unlock()
}

func (c *httpClient) EnableCookies() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.hasCookieJar {
		return
	}
	jar, _ := cookiejar.New(nil)
	c.underlying.Jar = jar
	c.hasCookieJar = true
}

func (c *httpClient) RequireHTTPS(v bool) {
	c.mu.Lock()
	c.requireHTTPS = v
	c.rebuildTransport()
	c.mu.Unlock()
}

func (c *httpClient) SetTLSConfig(cfg *tls.Config) {
	c.mu.Lock()
	c.tlsConfig = cfg
	c.rebuildTransport()
	c.mu.Unlock()
}

func (c *httpClient) SetMaxResponseSize(n int64) {
	c.mu.Lock()
	c.maxResponseSize = n
	c.mu.Unlock()
}

func (c *httpClient) SetValidateHost(fn func(string) bool) {
	c.mu.Lock()
	c.validateHost = fn
	c.rebuildTransport()
	c.mu.Unlock()
}

func (c *httpClient) SetValidateStatus(fn func(int) bool) {
	c.mu.Lock()
	c.validateStatus = fn
	c.mu.Unlock()
}

func (c *httpClient) OnBeforeRequest(fn func(*http.Request) error) {
	c.mu.Lock()
	c.beforeRequest = append(c.beforeRequest, fn)
	c.mu.Unlock()
}

func (c *httpClient) OnAfterResponse(fn func(*Response) error) {
	c.mu.Lock()
	c.afterResponse = append(c.afterResponse, fn)
	c.mu.Unlock()
}

func (c *httpClient) OnError(fn func(error)) {
	c.mu.Lock()
	c.onError = append(c.onError, fn)
	c.mu.Unlock()
}

func (c *httpClient) Download(rawURL, filepath string) error {
	return c.DownloadWithProgress(rawURL, filepath, nil)
}

func (c *httpClient) DownloadWithProgress(rawURL, filepath string, fn func(Progress)) error {
	resp, err := c.Get(rawURL).Stream().Send()
	if err != nil {
		return err
	}
	defer func() { _ = resp.BodyStream.Close() }()

	var total int64
	if resp.Raw != nil {
		total = resp.Raw.ContentLength
	}

	var r io.Reader = resp.BodyStream
	if fn != nil {
		r = &progressReader{r: resp.BodyStream, total: total, onProgress: fn}
	}
	return saveToFile(r, filepath)
}

// buildFinalHandler returns the terminal Handler that executes the HTTP request
// and reads the response body.
func (c *httpClient) buildFinalHandler(tc *timingCollector, isStream bool, onProgress func(Progress)) Handler {
	return func(req *http.Request) (*Response, error) {
		c.mu.RLock()
		underlying := c.underlying
		maxSize := c.maxResponseSize
		c.mu.RUnlock()

		raw, err := underlying.Do(req)
		if err != nil {
			return nil, &Error{Request: req, Cause: err}
		}

		resp := &Response{
			StatusCode: raw.StatusCode,
			Headers:    raw.Header,
			Raw:        raw,
		}

		total := raw.ContentLength

		if isStream {
			if onProgress != nil {
				resp.BodyStream = &progressReadCloser{
					ReadCloser: raw.Body,
					total:      total,
					onProgress: onProgress,
				}
			} else {
				resp.BodyStream = raw.Body
			}
		} else {
			defer func() { _ = raw.Body.Close() }()

			limited := io.LimitReader(raw.Body, maxSize+1)
			reader := limited
			if onProgress != nil {
				reader = &progressReader{r: limited, total: total, onProgress: onProgress}
			}

			data, readErr := io.ReadAll(reader)
			if readErr != nil {
				return nil, &Error{Request: req, Cause: readErr}
			}
			if int64(len(data)) > maxSize {
				return nil, &Error{Request: req, Cause: fmt.Errorf(
					"httpclient: response size exceeds limit of %d bytes", maxSize)}
			}
			resp.Body = data
		}

		resp.Timing = tc.build()
		return resp, nil
	}
}
