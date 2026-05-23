package httpclient

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func BenchmarkGet(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"status":"ok"}`)
	}))
	defer srv.Close()

	c := newHTTPClient(&HttpClientModuleOptions{BaseURL: srv.URL})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get("/").Send()
	}
}

func BenchmarkPost_JSON(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	c := newHTTPClient(&HttpClientModuleOptions{BaseURL: srv.URL})
	payload := map[string]string{"name": "ginject", "email": "test@example.com"}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = c.Post("/").JSON(payload).Send()
	}
}

func BenchmarkMiddlewareChain_3(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	noop := func(next Handler) Handler {
		return func(req *http.Request) (*Response, error) { return next(req) }
	}

	c := newHTTPClient(&HttpClientModuleOptions{BaseURL: srv.URL})
	c.Use(noop, noop, noop)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get("/").Send()
	}
}

func BenchmarkQueryParams_5(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := newHTTPClient(&HttpClientModuleOptions{BaseURL: srv.URL})
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = c.Get("/").
			Query("page", 1).
			Query("limit", 20).
			Query("sort", "name").
			Query("order", "asc").
			Query("filter", "active").
			Send()
	}
}

func BenchmarkSSEReader(b *testing.B) {
	const raw = "id:1\nevent:update\ndata:hello\ndata:world\n\nid:2\ndata:bye\n\n"
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sr := NewSSEReader(newRepeatReader(raw, 1))
		for {
			_, ok := sr.Next()
			if !ok {
				break
			}
		}
	}
}

// repeatReader returns a Reader that emits s exactly n times.
type repeatReader struct {
	data []byte
	n    int
	pos  int
	done int
}

func newRepeatReader(s string, n int) *repeatReader {
	return &repeatReader{data: []byte(s), n: n}
}

func (r *repeatReader) Read(p []byte) (int, error) {
	if r.done >= r.n {
		return 0, io.EOF
	}
	if r.pos >= len(r.data) {
		r.done++
		if r.done >= r.n {
			return 0, io.EOF
		}
		r.pos = 0
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
