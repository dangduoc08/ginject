package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

func newBenchContext(method, origin string) *ctx.Context {
	req := httptest.NewRequest(method, "/", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	c := ctx.NewContext()
	c.Request = req
	c.ResponseWriter = rec
	c.Event = ctx.NewEvent()
	return c
}

func BenchmarkCORS_Use_StarOrigin(b *testing.B) {
	cors := CORS{AllowOrigin: "*"}
	mw := cors.NewMiddleware()
	c := newBenchContext(http.MethodGet, "https://example.com")
	b.ResetTimer()
	for range b.N {
		mw.Use(c, noop)
	}
}

func BenchmarkCORS_Use_NilOriginDefault(b *testing.B) {
	cors := CORS{}
	mw := cors.NewMiddleware()
	c := newBenchContext(http.MethodGet, "https://example.com")
	b.ResetTimer()
	for range b.N {
		mw.Use(c, noop)
	}
}

func BenchmarkCORS_Use_OriginMap(b *testing.B) {
	cors := CORS{AllowOrigin: []string{"https://example.com", "https://foo.com"}}
	mw := cors.NewMiddleware()
	c := newBenchContext(http.MethodGet, "https://example.com")
	b.ResetTimer()
	for range b.N {
		mw.Use(c, noop)
	}
}

func BenchmarkCORS_Use_Preflight(b *testing.B) {
	cors := CORS{}
	mw := cors.NewMiddleware()
	c := newBenchContext(http.MethodOptions, "https://example.com")
	c.Next = noop
	b.ResetTimer()
	for range b.N {
		mw.Use(c, noop)
	}
}

func BenchmarkLoadCORSOptions(b *testing.B) {
	cors := &CORS{AllowOrigin: []string{"https://example.com", "https://foo.com"}}
	b.ResetTimer()
	for range b.N {
		loadCORSOptions(cors)
	}
}
