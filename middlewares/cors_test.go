package middlewares

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

func newTestContext(method, origin string) (*ctx.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, "/", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	c := ctx.NewContext()
	c.Request = req
	c.ResponseWriter = rec
	c.Event = ctx.NewEvent()
	return c, rec
}

func noop() {}

func TestLoadCORSOptions_NilAllowOriginDefaultsToStar(t *testing.T) {
	cors := &CORS{}
	opts := loadCORSOptions(cors)
	if opts.allowOrigin != "*" {
		t.Error(testutils.DiffMessage(opts.allowOrigin, "*", "nil AllowOrigin should default to *"))
	}
}

func TestLoadCORSOptions_SliceAllowOriginConvertsToMap(t *testing.T) {
	cors := &CORS{AllowOrigin: []string{"https://example.com", "https://foo.com"}}
	opts := loadCORSOptions(cors)
	m, ok := opts.allowOrigin.(map[string]bool)
	if !ok {
		t.Error(testutils.DiffMessage(opts.allowOrigin, "map[string]bool", "[]string should convert to map"))
		return
	}
	if !m["https://example.com"] {
		t.Error(testutils.DiffMessage(false, true, "https://example.com should be in map"))
	}
}

func TestLoadCORSOptions_StringAllowOriginPassesThrough(t *testing.T) {
	cors := &CORS{AllowOrigin: "https://example.com"}
	opts := loadCORSOptions(cors)
	if opts.allowOrigin != "https://example.com" {
		t.Error(testutils.DiffMessage(opts.allowOrigin, "https://example.com", "string AllowOrigin"))
	}
}

func TestLoadCORSOptions_RegexpPassesThrough(t *testing.T) {
	re := regexp.MustCompile(`^https://.*\.example\.com$`)
	cors := &CORS{AllowOrigin: re}
	opts := loadCORSOptions(cors)
	if opts.allowOrigin != re {
		t.Error(testutils.DiffMessage(opts.allowOrigin, re, "regexp AllowOrigin should pass through"))
	}
}

func TestLoadCORSOptions_DefaultMaxAge(t *testing.T) {
	cors := &CORS{}
	opts := loadCORSOptions(cors)
	if opts.maxAge != "5" {
		t.Error(testutils.DiffMessage(opts.maxAge, "5", "default MaxAge should be 5s"))
	}
}

func TestLoadCORSOptions_CustomMaxAge(t *testing.T) {
	cors := &CORS{MaxAge: 10000}
	opts := loadCORSOptions(cors)
	if opts.maxAge != "10" {
		t.Error(testutils.DiffMessage(opts.maxAge, "10", "MaxAge=10000ms should give 10s"))
	}
}

func TestLoadCORSOptions_DefaultSuccessStatus(t *testing.T) {
	cors := &CORS{}
	opts := loadCORSOptions(cors)
	if opts.optionsSuccessStatus != 204 {
		t.Error(testutils.DiffMessage(opts.optionsSuccessStatus, 204, "default status 204"))
	}
}

func TestLoadCORSOptions_DefaultAllowMethods(t *testing.T) {
	cors := &CORS{}
	opts := loadCORSOptions(cors)
	want := "GET, HEAD, PUT, PATCH, POST, DELETE"
	if opts.allowMethods != want {
		t.Error(testutils.DiffMessage(opts.allowMethods, want, "default allow methods"))
	}
}

func TestLoadCORSOptions_AllowHeadersSlice(t *testing.T) {
	cors := &CORS{AllowHeaders: []string{"Content-Type", "Authorization"}}
	opts := loadCORSOptions(cors)
	if opts.allowHeaders != "Content-Type, Authorization" {
		t.Error(testutils.DiffMessage(opts.allowHeaders, "Content-Type, Authorization", "allowHeaders joined"))
	}
}

func TestLoadCORSOptions_ExposeHeadersSlice(t *testing.T) {
	cors := &CORS{ExposeHeaders: []string{"X-Custom-Header"}}
	opts := loadCORSOptions(cors)
	if opts.exposeHeaders != "X-Custom-Header" {
		t.Error(testutils.DiffMessage(opts.exposeHeaders, "X-Custom-Header", "exposeHeaders joined"))
	}
}

func TestCORS_Use_SetsOriginStarByDefault(t *testing.T) {
	cors := CORS{}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "*" {
		t.Error(testutils.DiffMessage(got, "*", "default AllowOrigin should set * header"))
	}
}

func TestCORS_Use_SpecificOriginMap(t *testing.T) {
	cors := CORS{AllowOrigin: []string{"https://example.com"}}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "https://example.com" {
		t.Error(testutils.DiffMessage(got, "https://example.com", "allowed origin should echo"))
	}
}

func TestCORS_Use_SpecificOriginMapBlocked(t *testing.T) {
	cors := CORS{AllowOrigin: []string{"https://example.com"}}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://evil.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "" {
		t.Error(testutils.DiffMessage(got, "", "disallowed origin should not set header"))
	}
}

func TestCORS_Use_RegexpOrigin(t *testing.T) {
	cors := CORS{AllowOrigin: regexp.MustCompile(`^https://.*\.example\.com$`)}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://sub.example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "https://sub.example.com" {
		t.Error(testutils.DiffMessage(got, "https://sub.example.com", "regex-matched origin should echo"))
	}
}

func TestCORS_Use_Credentials(t *testing.T) {
	cors := CORS{IsAllowCredentials: true}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Credentials")
	if got != "true" {
		t.Error(testutils.DiffMessage(got, "true", "credentials header"))
	}
}

func TestCORS_Use_OptionsPreflightContinue(t *testing.T) {
	called := false
	cors := CORS{IsPreflightContinue: true}
	mw := cors.NewMiddleware()
	c, _ := newTestContext(http.MethodOptions, "https://example.com")
	c.Next = func() { called = true }
	mw.Use(c, noop)
	if !called {
		t.Error(testutils.DiffMessage(called, true, "IsPreflightContinue should call Next"))
	}
}

func TestCORS_Use_OptionsPreflightStatus(t *testing.T) {
	cors := CORS{IsPreflightContinue: false}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodOptions, "https://example.com")
	mw.Use(c, noop)
	if rec.Code != 204 {
		t.Error(testutils.DiffMessage(rec.Code, 204, "preflight status should be 204"))
	}
}

func TestCORS_Use_CustomOptionsSuccessStatus(t *testing.T) {
	cors := CORS{OptionsSuccessStatus: 200}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodOptions, "https://example.com")
	mw.Use(c, noop)
	if rec.Code != 200 {
		t.Error(testutils.DiffMessage(rec.Code, 200, "custom options success status"))
	}
}

func TestCORS_Use_NextCalledForNonOptions(t *testing.T) {
	called := false
	cors := CORS{}
	mw := cors.NewMiddleware()
	c, _ := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "next should be called for non-OPTIONS requests"))
	}
}

func TestCORS_Use_AllowHeadersString(t *testing.T) {
	cors := CORS{AllowHeaders: "Content-Type, Authorization"}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodOptions, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Headers")
	if got != "Content-Type, Authorization" {
		t.Error(testutils.DiffMessage(got, "Content-Type, Authorization", "string AllowHeaders"))
	}
}

func TestCORS_Use_ExposeHeadersString(t *testing.T) {
	cors := CORS{ExposeHeaders: "X-Custom-Header"}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Expose-Headers")
	if got != "X-Custom-Header" {
		t.Error(testutils.DiffMessage(got, "X-Custom-Header", "string ExposeHeaders"))
	}
}

func TestCORS_Use_CredentialsWithWildcardEchosOrigin(t *testing.T) {
	cors := CORS{AllowOrigin: "*", IsAllowCredentials: true}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "https://example.com" {
		t.Error(testutils.DiffMessage(got, "https://example.com", "credentials+wildcard should echo request origin"))
	}
	if rec.Header().Get("Vary") == "" {
		t.Error(testutils.DiffMessage("", "Vary", "Vary header required when echoing origin"))
	}
}

func TestCORS_Use_VaryForSpecificStringOrigin(t *testing.T) {
	cors := CORS{AllowOrigin: "https://example.com"}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	if rec.Header().Get("Vary") == "" {
		t.Error(testutils.DiffMessage("", "Origin", "specific string origin should set Vary: Origin"))
	}
}

func TestCORS_Use_NoVaryForWildcard(t *testing.T) {
	cors := CORS{AllowOrigin: "*"}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	if rec.Header().Get("Vary") != "" {
		t.Error(testutils.DiffMessage(rec.Header().Get("Vary"), "", "wildcard origin should not set Vary"))
	}
}

func TestCORS_Use_NoOriginHeaderSkipsCORS(t *testing.T) {
	cors := CORS{}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "next should be called when no Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error(testutils.DiffMessage(rec.Header().Get("Access-Control-Allow-Origin"), "", "no CORS headers without Origin"))
	}
}

func TestLoadCORSOptions_EmptySliceAllowOriginGivesEmptyMap(t *testing.T) {
	cors := &CORS{AllowOrigin: []string{}}
	opts := loadCORSOptions(cors)
	m, ok := opts.allowOrigin.(map[string]bool)
	if !ok {
		t.Error(testutils.DiffMessage(opts.allowOrigin, "map[string]bool", "empty []string should still convert to map"))
		return
	}
	if len(m) != 0 {
		t.Error(testutils.DiffMessage(len(m), 0, "empty slice should produce empty map"))
	}
}

func TestLoadCORSOptions_CustomAllowMethods(t *testing.T) {
	cors := &CORS{AllowMethods: []string{"GET", "POST"}}
	opts := loadCORSOptions(cors)
	if opts.allowMethods != "GET, POST" {
		t.Error(testutils.DiffMessage(opts.allowMethods, "GET, POST", "custom methods should be joined"))
	}
}

func TestLoadCORSOptions_OriginTrailingSlashTrimmed(t *testing.T) {
	cors := &CORS{AllowOrigin: []string{"https://example.com/"}}
	opts := loadCORSOptions(cors)
	m := opts.allowOrigin.(map[string]bool)
	if !m["https://example.com"] {
		t.Error(testutils.DiffMessage(false, true, "trailing slash should be trimmed from configured origin"))
	}
	if m["https://example.com/"] {
		t.Error(testutils.DiffMessage(true, false, "origin with trailing slash should not remain in map"))
	}
}

func TestCORS_Use_EmptySliceBlocksAllOrigins(t *testing.T) {
	cors := CORS{AllowOrigin: []string{}}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "" {
		t.Error(testutils.DiffMessage(got, "", "empty AllowOrigin list should block all origins"))
	}
}

func TestCORS_Use_CustomAllowMethodsOnPreflight(t *testing.T) {
	cors := CORS{AllowMethods: []string{"GET", "POST"}}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodOptions, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Methods")
	if got != "GET, POST" {
		t.Error(testutils.DiffMessage(got, "GET, POST", "custom AllowMethods should appear on preflight"))
	}
}

func TestCORS_Use_OriginTrailingSlashMatchesRequest(t *testing.T) {
	cors := CORS{AllowOrigin: []string{"https://example.com/"}}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "https://example.com" {
		t.Error(testutils.DiffMessage(got, "https://example.com", "configured origin with trailing slash should match bare request origin"))
	}
}

func TestCORS_Use_RegexpOriginNoMatch(t *testing.T) {
	cors := CORS{AllowOrigin: regexp.MustCompile(`^https://trusted\.com$`)}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "https://evil.com")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "" {
		t.Error(testutils.DiffMessage(got, "", "non-matching regexp origin should not set ACAO header"))
	}
}

func TestCORS_Use_NullOriginWithCredentialsBlocked(t *testing.T) {
	cors := CORS{AllowOrigin: "*", IsAllowCredentials: true}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "null")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got == "null" {
		t.Error(testutils.DiffMessage(got, "", "null origin must not be reflected when credentials enabled"))
	}
}

func TestCORS_Use_NullOriginWildcardNoCredentials(t *testing.T) {
	cors := CORS{AllowOrigin: "*"}
	mw := cors.NewMiddleware()
	c, rec := newTestContext(http.MethodGet, "null")
	mw.Use(c, noop)
	got := rec.Header().Get("Access-Control-Allow-Origin")
	if got != "*" {
		t.Error(testutils.DiffMessage(got, "*", "null origin without credentials: wildcard * should still be set"))
	}
}

func TestCORS_Use_PreflightOnlyHeaders(t *testing.T) {
	cors := CORS{}
	mw := cors.NewMiddleware()

	cGet, recGet := newTestContext(http.MethodGet, "https://example.com")
	mw.Use(cGet, noop)
	if recGet.Header().Get("Access-Control-Allow-Methods") != "" {
		t.Error(testutils.DiffMessage(recGet.Header().Get("Access-Control-Allow-Methods"), "", "Allow-Methods should not be set on non-preflight"))
	}
	if recGet.Header().Get("Access-Control-Max-Age") != "" {
		t.Error(testutils.DiffMessage(recGet.Header().Get("Access-Control-Max-Age"), "", "Max-Age should not be set on non-preflight"))
	}

	cOpt, recOpt := newTestContext(http.MethodOptions, "https://example.com")
	mw.Use(cOpt, noop)
	if recOpt.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error(testutils.DiffMessage("", "non-empty", "Allow-Methods should be set on preflight"))
	}
	if recOpt.Header().Get("Access-Control-Max-Age") == "" {
		t.Error(testutils.DiffMessage("", "non-empty", "Max-Age should be set on preflight"))
	}
}
