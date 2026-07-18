package helmet

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func noop() {}

func newTestContext(method, origin string) (*ctx.HTTPContext, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, "/", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	c := ctx.NewHTTPContext()
	c.Request = req
	c.ResponseWriter = rec
	c.Broker = broker.NewWithConfig(broker.Config{RecoverPanics: true})
	return c, rec
}

// loadHelmetOptions tests

func TestLoadHelmetOptions_DefaultCSP(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.contentSecurityPolicy != helmetDefaultCSP {
		t.Error(test.DiffMessage(opts.contentSecurityPolicy, helmetDefaultCSP, "default CSP"))
	}
}

func TestLoadHelmetOptions_CustomCSP(t *testing.T) {
	csp := "default-src 'none'"
	opts := loadHelmetOptions(&Helmet{ContentSecurityPolicy: csp})
	if opts.contentSecurityPolicy != csp {
		t.Error(test.DiffMessage(opts.contentSecurityPolicy, csp, "custom CSP"))
	}
}

func TestLoadHelmetOptions_DefaultCrossOriginEmbedderPolicy(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.crossOriginEmbedderPolicy != "require-corp" {
		t.Error(test.DiffMessage(opts.crossOriginEmbedderPolicy, "require-corp", "default COEP"))
	}
}

func TestLoadHelmetOptions_DefaultCrossOriginOpenerPolicy(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.crossOriginOpenerPolicy != "same-origin" {
		t.Error(test.DiffMessage(opts.crossOriginOpenerPolicy, "same-origin", "default COOP"))
	}
}

func TestLoadHelmetOptions_DefaultCrossOriginResourcePolicy(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.crossOriginResourcePolicy != "same-origin" {
		t.Error(test.DiffMessage(opts.crossOriginResourcePolicy, "same-origin", "default CORP"))
	}
}

func TestLoadHelmetOptions_DefaultDNSPrefetchControl(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.dnsPrefetchControl != "off" {
		t.Error(test.DiffMessage(opts.dnsPrefetchControl, "off", "default DNS prefetch control"))
	}
}

func TestLoadHelmetOptions_DefaultFrameOptions(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.frameOptions != "SAMEORIGIN" {
		t.Error(test.DiffMessage(opts.frameOptions, "SAMEORIGIN", "default frame options"))
	}
}

func TestLoadHelmetOptions_DefaultPermittedCrossDomainPolicies(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.permittedCrossDomainPolicies != "none" {
		t.Error(test.DiffMessage(opts.permittedCrossDomainPolicies, "none", "default permitted cross domain policies"))
	}
}

func TestLoadHelmetOptions_DefaultReferrerPolicy(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	if opts.referrerPolicy != "no-referrer" {
		t.Error(test.DiffMessage(opts.referrerPolicy, "no-referrer", "default referrer policy"))
	}
}

func TestLoadHelmetOptions_DefaultHSTS(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{})
	want := "max-age=15552000; includeSubDomains"
	if opts.hsts != want {
		t.Error(test.DiffMessage(opts.hsts, want, "default HSTS"))
	}
}

func TestLoadHelmetOptions_CustomHSTSMaxAge(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{HSTSMaxAge: 3600})
	if !strings.HasPrefix(opts.hsts, "max-age=3600") {
		t.Error(test.DiffMessage(opts.hsts, "max-age=3600...", "custom HSTS max-age"))
	}
}

func TestLoadHelmetOptions_HSTSDisabledByFlag(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{DisableHSTS: true})
	if opts.hsts != "" {
		t.Error(test.DiffMessage(opts.hsts, "", "HSTS should be disabled"))
	}
}

func TestLoadHelmetOptions_HSTSExcludeSubDomains(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{HSTSExcludeSubDomains: true})
	if strings.Contains(opts.hsts, "includeSubDomains") {
		t.Error(test.DiffMessage(opts.hsts, "no includeSubDomains", "HSTS should exclude subdomains"))
	}
	if !strings.HasPrefix(opts.hsts, "max-age=") {
		t.Error(test.DiffMessage(opts.hsts, "max-age=...", "HSTS max-age still required"))
	}
}

func TestLoadHelmetOptions_HSTSPreload(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{HSTSPreload: true})
	if !strings.HasSuffix(opts.hsts, "; preload") {
		t.Error(test.DiffMessage(opts.hsts, "...preload", "HSTS preload missing"))
	}
}

func TestLoadHelmetOptions_HSTSCustomMaxAgeAndSubDomains(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{HSTSMaxAge: 86400, HSTSExcludeSubDomains: true, HSTSPreload: true})
	want := "max-age=86400; preload"
	if opts.hsts != want {
		t.Error(test.DiffMessage(opts.hsts, want, "HSTS with custom max-age, no subdomains, preload"))
	}
}

func TestLoadHelmetOptions_CustomCrossOriginPolicies(t *testing.T) {
	opts := loadHelmetOptions(&Helmet{
		CrossOriginEmbedderPolicy: "unsafe-none",
		CrossOriginOpenerPolicy:   "same-origin-allow-popups",
		CrossOriginResourcePolicy: "cross-origin",
	})
	if opts.crossOriginEmbedderPolicy != "unsafe-none" {
		t.Error(test.DiffMessage(opts.crossOriginEmbedderPolicy, "unsafe-none", "custom COEP"))
	}
	if opts.crossOriginOpenerPolicy != "same-origin-allow-popups" {
		t.Error(test.DiffMessage(opts.crossOriginOpenerPolicy, "same-origin-allow-popups", "custom COOP"))
	}
	if opts.crossOriginResourcePolicy != "cross-origin" {
		t.Error(test.DiffMessage(opts.crossOriginResourcePolicy, "cross-origin", "custom CORP"))
	}
}

// Use tests

func TestHelmet_Use_CallsNext(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, _ := newTestContext(http.MethodGet, "")
	called := false
	mw.Use(c.Request, c.ResponseWriter, func() { called = true })
	if !called {
		t.Error(test.DiffMessage(called, true, "next should always be called"))
	}
}

func TestHelmet_Use_SetsXContentTypeOptions(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error(test.DiffMessage(rec.Header().Get("X-Content-Type-Options"), "nosniff", "X-Content-Type-Options"))
	}
}

func TestHelmet_Use_SetsXDownloadOptions(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("X-Download-Options") != "noopen" {
		t.Error(test.DiffMessage(rec.Header().Get("X-Download-Options"), "noopen", "X-Download-Options"))
	}
}

func TestHelmet_Use_SetsXXSSProtectionToZero(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("X-XSS-Protection") != "0" {
		t.Error(test.DiffMessage(rec.Header().Get("X-XSS-Protection"), "0", "X-XSS-Protection must be 0"))
	}
}

func TestHelmet_Use_SetsOriginAgentCluster(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("Origin-Agent-Cluster") != "?1" {
		t.Error(test.DiffMessage(rec.Header().Get("Origin-Agent-Cluster"), "?1", "Origin-Agent-Cluster"))
	}
}

func TestHelmet_Use_SetsDefaultCSP(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("Content-Security-Policy") != helmetDefaultCSP {
		t.Error(test.DiffMessage(rec.Header().Get("Content-Security-Policy"), helmetDefaultCSP, "default CSP header"))
	}
}

func TestHelmet_Use_SetsCustomCSP(t *testing.T) {
	csp := "default-src 'none'"
	mw := Helmet{ContentSecurityPolicy: csp}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("Content-Security-Policy") != csp {
		t.Error(test.DiffMessage(rec.Header().Get("Content-Security-Policy"), csp, "custom CSP header"))
	}
}

func TestHelmet_Use_SetsDefaultHSTS(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	want := "max-age=15552000; includeSubDomains"
	if rec.Header().Get("Strict-Transport-Security") != want {
		t.Error(test.DiffMessage(rec.Header().Get("Strict-Transport-Security"), want, "default HSTS header"))
	}
}

func TestHelmet_Use_SkipsHSTSWhenDisabled(t *testing.T) {
	mw := Helmet{DisableHSTS: true}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error(test.DiffMessage(rec.Header().Get("Strict-Transport-Security"), "", "HSTS should not be set"))
	}
}

func TestHelmet_Use_SetsDefaultFrameOptions(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Error(test.DiffMessage(rec.Header().Get("X-Frame-Options"), "SAMEORIGIN", "X-Frame-Options"))
	}
}

func TestHelmet_Use_SetsDefaultReferrerPolicy(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("Referrer-Policy") != "no-referrer" {
		t.Error(test.DiffMessage(rec.Header().Get("Referrer-Policy"), "no-referrer", "Referrer-Policy"))
	}
}

func TestHelmet_Use_SetsCrossOriginHeaders(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("Cross-Origin-Embedder-Policy") != "require-corp" {
		t.Error(test.DiffMessage(rec.Header().Get("Cross-Origin-Embedder-Policy"), "require-corp", "COEP"))
	}
	if rec.Header().Get("Cross-Origin-Opener-Policy") != "same-origin" {
		t.Error(test.DiffMessage(rec.Header().Get("Cross-Origin-Opener-Policy"), "same-origin", "COOP"))
	}
	if rec.Header().Get("Cross-Origin-Resource-Policy") != "same-origin" {
		t.Error(test.DiffMessage(rec.Header().Get("Cross-Origin-Resource-Policy"), "same-origin", "CORP"))
	}
}

func TestHelmet_Use_SetsCustomFrameOptions(t *testing.T) {
	mw := Helmet{FrameOptions: "DENY"}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Error(test.DiffMessage(rec.Header().Get("X-Frame-Options"), "DENY", "custom frame options"))
	}
}

func TestHelmet_Use_SetsCustomReferrerPolicy(t *testing.T) {
	mw := Helmet{ReferrerPolicy: "strict-origin"}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("Referrer-Policy") != "strict-origin" {
		t.Error(test.DiffMessage(rec.Header().Get("Referrer-Policy"), "strict-origin", "custom referrer policy"))
	}
}

func TestHelmet_Use_SetsDefaultDNSPrefetchControl(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("X-DNS-Prefetch-Control") != "off" {
		t.Error(test.DiffMessage(rec.Header().Get("X-DNS-Prefetch-Control"), "off", "X-DNS-Prefetch-Control"))
	}
}

func TestHelmet_Use_SetsDefaultPermittedCrossDomainPolicies(t *testing.T) {
	mw := Helmet{}.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	mw.Use(c.Request, c.ResponseWriter, noop)
	if rec.Header().Get("X-Permitted-Cross-Domain-Policies") != "none" {
		t.Error(test.DiffMessage(rec.Header().Get("X-Permitted-Cross-Domain-Policies"), "none", "X-Permitted-Cross-Domain-Policies"))
	}
}
