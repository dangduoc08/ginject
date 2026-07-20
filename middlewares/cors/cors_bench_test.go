package cors

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

func newBenchContext(method, origin string) *ctx.HTTPContext {
	req := httptest.NewRequest(method, "/", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	c := ctx.NewHTTPContext()
	c.Request = req
	c.ResponseWriter = rec
	return c
}

// benchUseCORS measures Use() in isolation: the request and broker are built
// once and reused (CORS never mutates either), but the response recorder -
// the only state CORS actually writes to - is fresh every iteration, exactly
// like every real request gets its own response header map.
func benchUseCORS(b *testing.B, cors CORS, method string) {
	mw := cors.NewMiddleware()
	c := newBenchContext(method, "https://example.com")
	b.ResetTimer()
	for range b.N {
		c.ResponseWriter = httptest.NewRecorder()
		mw.Use(c.Request, c.ResponseWriter, noop)
	}
}

func BenchmarkCORS_Use_StarOrigin(b *testing.B) {
	benchUseCORS(b, CORS{AllowOrigin: "*"}, http.MethodGet)
}

func BenchmarkCORS_Use_NilOriginDefault(b *testing.B) {
	benchUseCORS(b, CORS{}, http.MethodGet)
}

func BenchmarkCORS_Use_OriginMap(b *testing.B) {
	benchUseCORS(b, CORS{AllowOrigin: []string{"https://example.com", "https://foo.com"}}, http.MethodGet)
}

func BenchmarkCORS_Use_Preflight(b *testing.B) {
	benchUseCORS(b, CORS{}, http.MethodOptions)
}

func BenchmarkMatchOrigin_Wildcard(b *testing.B) {
	opts := loadCORSOptions(&CORS{})
	b.ResetTimer()
	for range b.N {
		_, _ = matchOrigin(opts.allowOrigin, "https://example.com", opts.isAllowCredentials)
	}
}

func BenchmarkMatchOrigin_Map(b *testing.B) {
	opts := loadCORSOptions(&CORS{AllowOrigin: []string{"https://example.com", "https://foo.com"}})
	b.ResetTimer()
	for range b.N {
		_, _ = matchOrigin(opts.allowOrigin, "https://example.com", opts.isAllowCredentials)
	}
}

func BenchmarkMatchOrigin_Regexp(b *testing.B) {
	opts := loadCORSOptions(&CORS{AllowOrigin: regexp.MustCompile(`^https://.*\.example\.com$`)})
	b.ResetTimer()
	for range b.N {
		_, _ = matchOrigin(opts.allowOrigin, "https://sub.example.com", opts.isAllowCredentials)
	}
}

func BenchmarkLoadCORSOptions(b *testing.B) {
	cors := &CORS{AllowOrigin: []string{"https://example.com", "https://foo.com"}}
	b.ResetTimer()
	for range b.N {
		loadCORSOptions(cors)
	}
}
