package csrf

import (
	"net/http"
	"testing"
)

func BenchmarkGenerateCSRFToken(b *testing.B) {
	b.ResetTimer()
	for range b.N {
		_, _ = GenerateCSRFToken(32)
	}
}

func BenchmarkCompareTokensSecurely_Equal(b *testing.B) {
	tok, _ := GenerateCSRFToken(32)
	b.ResetTimer()
	for range b.N {
		CompareTokensSecurely(tok, tok)
	}
}

func BenchmarkCompareTokensSecurely_Unequal(b *testing.B) {
	a, _ := GenerateCSRFToken(32)
	c, _ := GenerateCSRFToken(32)
	b.ResetTimer()
	for range b.N {
		CompareTokensSecurely(a, c)
	}
}

func BenchmarkCSRF_SafeMethod_NoCookie(b *testing.B) {
	mw := CSRF{}.NewMiddleware()
	b.ResetTimer()
	for range b.N {
		c, _ := newCSRFContext(http.MethodGet, "", "")
		mw.Use(c.Request, c.ResponseWriter, noop)
	}
}

func BenchmarkCSRF_SafeMethod_WithCookie(b *testing.B) {
	mw := CSRF{}.NewMiddleware()
	b.ResetTimer()
	for range b.N {
		c, _ := newCSRFContext(http.MethodGet, "", "existingtoken123456")
		mw.Use(c.Request, c.ResponseWriter, noop)
	}
}

func BenchmarkCSRF_POST_ValidHeader(b *testing.B) {
	mw := CSRF{}.NewMiddleware()
	b.ResetTimer()
	for range b.N {
		c, _ := newCSRFContextWithHeader(http.MethodPost, csrfDefaultHeaderName, "tok", "tok")
		mw.Use(c.Request, c.ResponseWriter, noop)
	}
}

func BenchmarkLoadCSRFOptions(b *testing.B) {
	cfg := &CSRF{TokenLength: 32, CookieName: "_csrf", HeaderName: "X-CSRF-Token"}
	b.ResetTimer()
	for range b.N {
		loadCSRFOptions(cfg)
	}
}
