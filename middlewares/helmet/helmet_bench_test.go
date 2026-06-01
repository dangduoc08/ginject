package helmet

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/broker"
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
	c.Broker = broker.NewWithConfig(broker.Config{RecoverPanics: true})
	return c
}

func BenchmarkHelmet_Use_Defaults(b *testing.B) {
	mw := Helmet{}.NewMiddleware()
	c := newBenchContext(http.MethodGet, "")
	b.ResetTimer()
	for range b.N {
		mw.Use(c, noop)
	}
}

func BenchmarkHelmet_Use_CustomCSP(b *testing.B) {
	mw := Helmet{ContentSecurityPolicy: "default-src 'none'; img-src 'self'"}.NewMiddleware()
	c := newBenchContext(http.MethodGet, "")
	b.ResetTimer()
	for range b.N {
		mw.Use(c, noop)
	}
}

func BenchmarkHelmet_Use_DisableHSTS(b *testing.B) {
	mw := Helmet{DisableHSTS: true}.NewMiddleware()
	c := newBenchContext(http.MethodGet, "")
	b.ResetTimer()
	for range b.N {
		mw.Use(c, noop)
	}
}

func BenchmarkLoadHelmetOptions(b *testing.B) {
	h := &Helmet{
		ContentSecurityPolicy: "default-src 'self'",
		HSTSMaxAge:            86400,
		HSTSPreload:           true,
	}
	b.ResetTimer()
	for range b.N {
		loadHelmetOptions(h)
	}
}
