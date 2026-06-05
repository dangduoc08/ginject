package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
)

// helpers

func newTestClient(srv *httptest.Server) *httpClient {
	c := newHTTPClient(&HTTPClientModuleOptions{BaseURL: srv.URL})
	return c
}

func echoServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		w.Header().Set("X-Method", r.Method)
		w.Header().Set("X-Query", r.URL.RawQuery)
		for k, vv := range r.Header {
			for _, v := range vv {
				w.Header().Add("Echo-"+k, v)
			}
		}
		_, _ = w.Write(body)
	}))
}

func statusServer(code int, body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)
		_, _ = io.WriteString(w, body)
	}))
}

// --- basic HTTP methods ---

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "hello")
	}))
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resp.StatusCode, 200; got != want {
		t.Error(test.DiffMessage(got, want, "status code"))
	}
	if got, want := resp.Text(), "hello"; got != want {
		t.Error(test.DiffMessage(got, want, "body"))
	}
}

func TestPost_JSON(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	payload := map[string]string{"name": "ginject"}
	resp, err := c.Post("/").JSON(payload).Send()
	if err != nil {
		t.Fatal(err)
	}

	var got map[string]string
	if err := resp.JSON(&got); err != nil {
		t.Fatal(err)
	}
	if got["name"] != "ginject" {
		t.Error(test.DiffMessage(got["name"], "ginject", "json field name"))
	}
	ct := resp.Headers.Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Error(test.DiffMessage(ct, "application/json", "Content-Type"))
	}
}

func TestPut(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Put("/").JSON(map[string]int{"v": 1}).Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Headers.Get("X-Method") != http.MethodPut {
		t.Error(test.DiffMessage(resp.Headers.Get("X-Method"), "PUT", "method"))
	}
}

func TestDelete(t *testing.T) {
	srv := statusServer(204, "")
	defer srv.Close()

	c := newHTTPClient(&HTTPClientModuleOptions{
		BaseURL: srv.URL,
		// 204 is not in default 200-399 range? Yes it is (200 ≤ 204 < 400).
	})
	resp, err := c.Delete("/resource/1").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 204 {
		t.Error(test.DiffMessage(resp.StatusCode, 204, "status"))
	}
}

// --- query params ---

func TestQueryParams(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Get("/").
		Query("page", 2).
		Query("tags", []string{"go", "test"}).
		Send()
	if err != nil {
		t.Fatal(err)
	}
	q := resp.Headers.Get("X-Query")
	if !strings.Contains(q, "page=2") {
		t.Error(test.DiffMessage(q, "page=2", "query param page"))
	}
	if !strings.Contains(q, "tags=go") {
		t.Error(test.DiffMessage(q, "tags=go", "query param tags"))
	}
}

// --- headers ---

func TestCustomHeaders(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Get("/").Header("X-Custom", "hello").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Headers.Get("Echo-X-Custom") != "hello" {
		t.Error(test.DiffMessage(resp.Headers.Get("Echo-X-Custom"), "hello", "custom header"))
	}
}

func TestDefaultHeaders(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	c.SetHeader("X-App", "myapp")
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Headers.Get("Echo-X-App") != "myapp" {
		t.Error(test.DiffMessage(resp.Headers.Get("Echo-X-App"), "myapp", "default header"))
	}
}

func TestHeaderOverridesDefault(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	c.SetHeader("X-Version", "1")
	resp, err := c.Get("/").Header("X-Version", "2").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Headers.Get("Echo-X-Version") != "2" {
		t.Error(test.DiffMessage(resp.Headers.Get("Echo-X-Version"), "2", "header override"))
	}
}

// --- base URL ---

func TestBaseURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	}))
	defer srv.Close()

	c := newHTTPClient(&HTTPClientModuleOptions{BaseURL: srv.URL})
	resp, err := c.Get("/users/42").Send()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resp.Text(), "/users/42"; got != want {
		t.Error(test.DiffMessage(got, want, "path"))
	}
}

func TestFullURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	c := newHTTPClient(nil) // no base URL
	resp, err := c.Get(srv.URL + "/ping").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Error(test.DiffMessage(resp.StatusCode, 200, "full url status"))
	}
}

// --- middleware ---

func TestMiddleware(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	var called bool
	mw := func(next Handler) Handler {
		return func(req *http.Request) (*Response, error) {
			called = true
			return next(req)
		}
	}

	c := newTestClient(srv)
	c.Use(mw)
	_, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error(test.DiffMessage(called, true, "middleware called"))
	}
}

func TestMiddlewareChainOrder(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	var order []int
	mkMW := func(n int) Middleware {
		return func(next Handler) Handler {
			return func(req *http.Request) (*Response, error) {
				order = append(order, n)
				return next(req)
			}
		}
	}

	c := newTestClient(srv)
	c.Use(mkMW(1), mkMW(2), mkMW(3))
	_, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Error(test.DiffMessage(order, []int{1, 2, 3}, "middleware order"))
	}
}

// --- retry ---

func TestRetry_On500(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(500)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.SetRetry(3)
	c.SetRetryBackoff(1*time.Millisecond, 10*time.Millisecond)
	c.SetValidateStatus(func(code int) bool { return code < 500 })
	_, err := c.Get("/").Send()

	if err == nil {
		t.Error("expected error for status 500")
	}
	if got, want := int(atomic.LoadInt32(&count)), 4; got != want {
		t.Error(test.DiffMessage(got, want, "retry attempts (1 original + 3 retries)"))
	}
}

func TestRetry_SuccessOnThirdAttempt(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&count, 1)
		if n < 3 {
			w.WriteHeader(500)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.SetRetry(3)
	c.SetRetryBackoff(1*time.Millisecond, 10*time.Millisecond)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Error(test.DiffMessage(resp.StatusCode, 200, "status after retry"))
	}
	if got, want := int(atomic.LoadInt32(&count)), 3; got != want {
		t.Error(test.DiffMessage(got, want, "attempts"))
	}
}

func TestRetry_PerRequest(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&count, 1)
		w.WriteHeader(503)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.SetValidateStatus(func(code int) bool { return code < 500 })
	_, _ = c.Get("/").Retry(2).RetryBackoff(1*time.Millisecond, 5*time.Millisecond).Send()
	if got, want := int(atomic.LoadInt32(&count)), 3; got != want {
		t.Error(test.DiffMessage(got, want, "per-request retry attempts"))
	}
}

// --- validate status ---

func TestValidateStatus_DefaultRejects4xx(t *testing.T) {
	srv := statusServer(404, "not found")
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Get("/").Send()
	if err == nil {
		t.Error("expected error for 404")
	}
	var httpErr *Error
	if !errors.As(err, &httpErr) {
		t.Errorf("expected *Error, got %T", err)
	}
	if httpErr.Response.StatusCode != 404 {
		t.Error(test.DiffMessage(httpErr.Response.StatusCode, 404, "error response status"))
	}
}

func TestValidateStatus_Custom(t *testing.T) {
	srv := statusServer(404, "not found")
	defer srv.Close()

	c := newTestClient(srv)
	c.SetValidateStatus(func(code int) bool { return true })
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Error(test.DiffMessage(resp.StatusCode, 404, "status with custom validator"))
	}
}

// --- streaming ---

func TestStream_BodyNotBuffered(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.WriteString(w, "streamed")
	}))
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Get("/").Stream().Send()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.BodyStream.Close() }()

	if resp.Body != nil {
		t.Error(test.DiffMessage(resp.Body, nil, "Body should be nil in stream mode"))
	}
	if resp.BodyStream == nil {
		t.Error(test.DiffMessage(resp.BodyStream, "non-nil ReadCloser", "BodyStream"))
	}
	data, _ := io.ReadAll(resp.BodyStream)
	if string(data) != "streamed" {
		t.Error(test.DiffMessage(string(data), "streamed", "stream content"))
	}
}

// --- SSE reader ---

func TestSSEReader(t *testing.T) {
	raw := "id:1\nevent:update\ndata:hello\ndata:world\n\nid:2\ndata:bye\n\n"
	sr := NewSSEReader(strings.NewReader(raw))

	evt, ok := sr.Next()
	if !ok {
		t.Fatal("expected first event")
	}
	if got, want := evt.ID, "1"; got != want {
		t.Error(test.DiffMessage(got, want, "event 1 ID"))
	}
	if got, want := evt.Event, "update"; got != want {
		t.Error(test.DiffMessage(got, want, "event 1 type"))
	}
	if got, want := evt.Data, "hello\nworld"; got != want {
		t.Error(test.DiffMessage(got, want, "event 1 data"))
	}

	evt, ok = sr.Next()
	if !ok {
		t.Fatal("expected second event")
	}
	if got, want := evt.ID, "2"; got != want {
		t.Error(test.DiffMessage(got, want, "event 2 ID"))
	}
	if got, want := evt.Data, "bye"; got != want {
		t.Error(test.DiffMessage(got, want, "event 2 data"))
	}

	_, ok = sr.Next()
	if ok {
		t.Error("expected stream end")
	}
}

func TestSSEReader_Comment(t *testing.T) {
	raw := ": comment\ndata:val\n\n"
	sr := NewSSEReader(strings.NewReader(raw))
	evt, ok := sr.Next()
	if !ok || evt.Data != "val" {
		t.Error(test.DiffMessage(evt, "data=val", "SSE comment skipped"))
	}
}

func TestSSEReader_RetryField(t *testing.T) {
	raw := "retry:3000\ndata:x\n\n"
	sr := NewSSEReader(strings.NewReader(raw))
	evt, ok := sr.Next()
	if !ok {
		t.Fatal("expected event")
	}
	if evt.Retry != 3000 {
		t.Error(test.DiffMessage(evt.Retry, 3000, "retry field"))
	}
}

// --- download ---

func TestDownload(t *testing.T) {
	content := "file content"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, content)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	dst := filepath.Join(t.TempDir(), "out.txt")
	if err := c.Download(srv.URL+"/file", dst); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != content {
		t.Error(test.DiffMessage(string(data), content, "downloaded file content"))
	}
}

func TestDownloadWithProgress(t *testing.T) {
	content := bytes.Repeat([]byte("x"), 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1024")
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	dst := filepath.Join(t.TempDir(), "big.bin")
	var lastPct float64
	err := c.DownloadWithProgress(srv.URL+"/file", dst, func(p Progress) {
		lastPct = p.Percent
	})
	if err != nil {
		t.Fatal(err)
	}
	if lastPct == 0 {
		t.Error("expected non-zero progress percent")
	}
}

// --- hooks ---

func TestHook_BeforeRequest(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	var called bool
	c.OnBeforeRequest(func(req *http.Request) error {
		req.Header.Set("X-Hook", "injected")
		called = true
		return nil
	})
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("OnBeforeRequest not called")
	}
	if resp.Headers.Get("Echo-X-Hook") != "injected" {
		t.Error(test.DiffMessage(resp.Headers.Get("Echo-X-Hook"), "injected", "hook-injected header"))
	}
}

func TestHook_BeforeRequest_Error(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	c := newTestClient(srv)
	hookErr := errors.New("blocked by hook")
	c.OnBeforeRequest(func(_ *http.Request) error { return hookErr })
	_, err := c.Get("/").Send()
	if !errors.Is(err, hookErr) {
		t.Error(test.DiffMessage(err, hookErr, "before-request hook error"))
	}
}

func TestHook_AfterResponse(t *testing.T) {
	srv := statusServer(200, "data")
	defer srv.Close()

	c := newTestClient(srv)
	var seen int
	c.OnAfterResponse(func(resp *Response) error {
		seen = resp.StatusCode
		return nil
	})
	_, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if seen != 200 {
		t.Error(test.DiffMessage(seen, 200, "after-response hook status"))
	}
}

func TestHook_OnError(t *testing.T) {
	srv := statusServer(500, "err")
	defer srv.Close()

	c := newTestClient(srv)
	var seen error
	c.OnError(func(err error) { seen = err })
	_, _ = c.Get("/").Send()
	if seen == nil {
		t.Error("OnError not called on status 500")
	}
}

// --- max response size ---

func TestMaxResponseSize(t *testing.T) {
	content := bytes.Repeat([]byte("A"), 1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.SetMaxResponseSize(512)
	c.SetValidateStatus(func(int) bool { return true })
	_, err := c.Get("/").Send()
	if err == nil {
		t.Error("expected error when response exceeds max size")
	}
	if !strings.Contains(err.Error(), "exceeds limit") {
		t.Error(test.DiffMessage(err.Error(), "contains 'exceeds limit'", "error message"))
	}
}

// --- timeout ---

func TestTimeout_RequestLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = io.WriteString(w, "late")
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.SetValidateStatus(func(int) bool { return true })
	_, err := c.Get("/").Timeout(50 * time.Millisecond).Send()
	if err == nil {
		t.Error("expected timeout error")
	}
}

// --- SSRF protection ---

func TestSSRF_HostNotAllowed(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	c := newTestClient(srv)
	c.SetValidateHost(func(host string) bool { return false })
	_, err := c.Get("/").Send()
	if err == nil {
		t.Error("expected SSRF error")
	}
	if !strings.Contains(err.Error(), "host not allowed") {
		t.Error(test.DiffMessage(err.Error(), "host not allowed", "SSRF error message"))
	}
}

func TestSSRF_HostAllowed(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	c := newTestClient(srv)
	c.SetValidateHost(func(host string) bool { return host == "127.0.0.1" })
	_, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
}

// --- HTTPS enforcement ---

func TestRequireHTTPS_Rejects_HTTP(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	c := newTestClient(srv) // srv is http://
	c.RequireHTTPS(true)
	_, err := c.Get("/").Send()
	if err == nil {
		t.Error("expected error: HTTPS required")
	}
	if !strings.Contains(err.Error(), "HTTPS required") {
		t.Error(test.DiffMessage(err.Error(), "HTTPS required", "error message"))
	}
}

// --- cookie jar ---

func TestEnableCookies(t *testing.T) {
	var cookieHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/set" {
			http.SetCookie(w, &http.Cookie{Name: "sess", Value: "abc"})
			return
		}
		cookieHeader = r.Header.Get("Cookie")
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.EnableCookies()
	_, _ = c.Get("/set").Send()
	_, _ = c.Get("/check").Send()

	if !strings.Contains(cookieHeader, "sess=abc") {
		t.Error(test.DiffMessage(cookieHeader, "contains sess=abc", "cookie jar"))
	}
}

// --- form body ---

func TestForm_URLEncoded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		_, _ = io.WriteString(w, r.FormValue("name"))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Post("/").Form(map[string]string{"name": "ginject"}).Send()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resp.Text(), "ginject"; got != want {
		t.Error(test.DiffMessage(got, want, "form field"))
	}
}

// --- multipart ---

func TestMultipart_File(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		file, header, err := r.FormFile("avatar")
		if err != nil {
			w.WriteHeader(400)
			return
		}
		defer func() { _ = file.Close() }()
		_, _ = io.WriteString(w, header.Filename)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Post("/upload").
		File("avatar", "photo.jpg", strings.NewReader("imgdata")).
		Field("user", "alice").
		Send()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := resp.Text(), "photo.jpg"; got != want {
		t.Error(test.DiffMessage(got, want, "uploaded filename"))
	}
}

// --- JSON response binding ---

func TestResponse_JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"count": 42})
	}))
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]int
	if err := resp.JSON(&result); err != nil {
		t.Fatal(err)
	}
	if result["count"] != 42 {
		t.Error(test.DiffMessage(result["count"], 42, "json field count"))
	}
}

// --- context cancellation ---

func TestContext_Cancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = io.WriteString(w, "late")
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.SetValidateStatus(func(int) bool { return true })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := c.Get("/").Context(ctx).Send()
	if err == nil {
		t.Error("expected cancellation error")
	}
}

// --- timing ---

func TestTiming_Populated(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Get("/").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.Timing == nil {
		t.Fatal("expected timing info")
	}
	if resp.Timing.Total == 0 {
		t.Error("expected non-zero Total timing")
	}
}

// --- module registration ---

func TestModule_Register(t *testing.T) {
	m := Register(&HTTPClientModuleOptions{IsGlobal: true})
	if m == nil {
		t.Fatal("Register returned nil")
	}
	if !m.IsGlobal {
		t.Error(test.DiffMessage(m.IsGlobal, true, "IsGlobal"))
	}
}

func TestModule_RegisterNilOpts(t *testing.T) {
	m := Register(nil)
	if m == nil {
		t.Fatal("Register(nil) returned nil")
	}
}

// --- error wrapping ---

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("inner")
	e := &Error{Cause: cause}
	if !errors.Is(e, cause) {
		t.Error("Error.Unwrap should expose Cause")
	}
}

func TestError_Message_WithResponse(t *testing.T) {
	srv := statusServer(404, "nf")
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Get("/thing").Send()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Error(test.DiffMessage(err.Error(), "contains 404", "error message"))
	}
}

// --- nil / empty inputs ---

func TestGet_EmptyPath(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Get("").Send()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Error(test.DiffMessage(resp.StatusCode, 200, "empty path status"))
	}
}

func TestJSON_NilBody(t *testing.T) {
	srv := echoServer()
	defer srv.Close()

	c := newTestClient(srv)
	resp, err := c.Post("/").JSON(nil).Send()
	if err != nil {
		t.Fatal(err)
	}
	_ = resp
}

// --- concurrent safety ---

func TestConcurrentRequests(t *testing.T) {
	srv := statusServer(200, "ok")
	defer srv.Close()

	c := newTestClient(srv)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = c.Get("/").Send()
		}()
	}
	wg.Wait()
}
