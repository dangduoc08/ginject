package middlewares

import (
	"net/http"
	"testing"
)

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
