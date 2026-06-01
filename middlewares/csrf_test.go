package middlewares

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

func newCSRFContext(method, token, cookieToken string) (*ctx.Context, *httptest.ResponseRecorder) {
	body := ""
	contentType := ""
	if token != "" && method != http.MethodGet {
		form := url.Values{"_csrf": {token}}
		body = form.Encode()
		contentType = "application/x-www-form-urlencoded"
	}

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, "/", strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, "/", nil)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if cookieToken != "" {
		req.AddCookie(&http.Cookie{Name: "_csrf", Value: cookieToken})
	}

	rec := httptest.NewRecorder()
	c := ctx.NewContext()
	c.Request = req
	c.ResponseWriter = rec
	c.Broker = broker.NewWithConfig(broker.Config{RecoverPanics: true})
	return c, rec
}

func newCSRFContextWithHeader(method, headerName, headerToken, cookieToken string) (*ctx.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, "/", nil)
	if headerToken != "" {
		req.Header.Set(headerName, headerToken)
	}
	if cookieToken != "" {
		req.AddCookie(&http.Cookie{Name: "_csrf", Value: cookieToken})
	}
	rec := httptest.NewRecorder()
	c := ctx.NewContext()
	c.Request = req
	c.ResponseWriter = rec
	c.Broker = broker.NewWithConfig(broker.Config{RecoverPanics: true})
	return c, rec
}

// --- loadCSRFOptions defaults ---

func TestLoadCSRFOptions_Defaults(t *testing.T) {
	opts := loadCSRFOptions(&CSRF{})
	if opts.tokenLength != csrfDefaultTokenLength {
		t.Error(testutils.DiffMessage(opts.tokenLength, csrfDefaultTokenLength, "default token length"))
	}
	if opts.cookieName != csrfDefaultCookieName {
		t.Error(testutils.DiffMessage(opts.cookieName, csrfDefaultCookieName, "default cookie name"))
	}
	if opts.headerName != csrfDefaultHeaderName {
		t.Error(testutils.DiffMessage(opts.headerName, csrfDefaultHeaderName, "default header name"))
	}
	if opts.contextKey != csrfDefaultContextKey {
		t.Error(testutils.DiffMessage(opts.contextKey, csrfDefaultContextKey, "default context key"))
	}
}

func TestLoadCSRFOptions_ZeroTokenLengthUsesDefault(t *testing.T) {
	opts := loadCSRFOptions(&CSRF{TokenLength: 0})
	if opts.tokenLength != csrfDefaultTokenLength {
		t.Error(testutils.DiffMessage(opts.tokenLength, csrfDefaultTokenLength, "zero length must default"))
	}
}

func TestLoadCSRFOptions_NegativeTokenLengthUsesDefault(t *testing.T) {
	opts := loadCSRFOptions(&CSRF{TokenLength: -1})
	if opts.tokenLength != csrfDefaultTokenLength {
		t.Error(testutils.DiffMessage(opts.tokenLength, csrfDefaultTokenLength, "negative length must default"))
	}
}

func TestLoadCSRFOptions_CustomValues(t *testing.T) {
	opts := loadCSRFOptions(&CSRF{
		TokenLength: 64,
		CookieName:  "my_csrf",
		HeaderName:  "X-My-CSRF",
		ContextKey:  "my_key",
	})
	if opts.tokenLength != 64 {
		t.Error(testutils.DiffMessage(opts.tokenLength, 64, "custom token length"))
	}
	if opts.cookieName != "my_csrf" {
		t.Error(testutils.DiffMessage(opts.cookieName, "my_csrf", "custom cookie name"))
	}
	if opts.headerName != "X-My-CSRF" {
		t.Error(testutils.DiffMessage(opts.headerName, "X-My-CSRF", "custom header name"))
	}
}

// --- GenerateCSRFToken ---

func TestGenerateCSRFToken_Length(t *testing.T) {
	tok, err := GenerateCSRFToken(32)
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "must not error"))
	}
	if len(tok) != 64 {
		t.Error(testutils.DiffMessage(len(tok), 64, "32 bytes → 64 hex chars"))
	}
}

func TestGenerateCSRFToken_Uniqueness(t *testing.T) {
	a, _ := GenerateCSRFToken(32)
	b, _ := GenerateCSRFToken(32)
	if a == b {
		t.Error(testutils.DiffMessage(a, "<different>", "tokens must be unique"))
	}
}

func TestGenerateCSRFToken_ZeroLengthUsesDefault(t *testing.T) {
	tok, err := GenerateCSRFToken(0)
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "zero length must not error"))
	}
	if len(tok) != csrfDefaultTokenLength*2 {
		t.Error(testutils.DiffMessage(len(tok), csrfDefaultTokenLength*2, "zero length must use default"))
	}
}

func TestGenerateCSRFToken_OnlyHexChars(t *testing.T) {
	tok, _ := GenerateCSRFToken(32)
	for _, ch := range tok {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			t.Error(testutils.DiffMessage(string(ch), "hex char", "token must be lowercase hex"))
			return
		}
	}
}

// --- CompareTokensSecurely ---

func TestCompareTokensSecurely_Equal(t *testing.T) {
	if !CompareTokensSecurely("abc", "abc") {
		t.Error(testutils.DiffMessage(false, true, "equal tokens must match"))
	}
}

func TestCompareTokensSecurely_Unequal(t *testing.T) {
	if CompareTokensSecurely("abc", "xyz") {
		t.Error(testutils.DiffMessage(true, false, "unequal tokens must not match"))
	}
}

func TestCompareTokensSecurely_EmptyBothEqual(t *testing.T) {
	if !CompareTokensSecurely("", "") {
		t.Error(testutils.DiffMessage(false, true, "two empty strings must match"))
	}
}

func TestCompareTokensSecurely_OneEmpty(t *testing.T) {
	if CompareTokensSecurely("abc", "") {
		t.Error(testutils.DiffMessage(true, false, "non-empty vs empty must not match"))
	}
}

// --- Safe methods pass through ---

func TestCSRF_SafeMethod_GET(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContext(http.MethodGet, "", "")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "GET must pass through"))
	}
}

func TestCSRF_SafeMethod_HEAD(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContext(http.MethodHead, "", "")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "HEAD must pass through"))
	}
}

func TestCSRF_SafeMethod_OPTIONS(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContext(http.MethodOptions, "", "")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "OPTIONS must pass through"))
	}
}

// --- Cookie generation ---

func TestCSRF_SetsCookieWhenMissing(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, rec := newCSRFContext(http.MethodGet, "", "")
	mw.Use(c, noop)
	cookies := rec.Result().Cookies()
	found := false
	for _, ck := range cookies {
		if ck.Name == csrfDefaultCookieName && ck.Value != "" {
			found = true
		}
	}
	if !found {
		t.Error(testutils.DiffMessage(found, true, "must set CSRF cookie when missing"))
	}
}

func TestCSRF_ReusesExistingCookie(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, rec := newCSRFContext(http.MethodGet, "", "existingtoken")
	mw.Use(c, noop)
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == csrfDefaultCookieName {
			t.Error(testutils.DiffMessage("cookie set", "no new cookie", "must not overwrite existing cookie"))
		}
	}
}

// --- Token stored in request context ---

func TestCSRF_StoresTokenInContext(t *testing.T) {
	mw := CSRF{ContextKey: "csrf_token"}.NewMiddleware()
	c, _ := newCSRFContext(http.MethodGet, "", "mytoken")
	mw.Use(c, func() {
		val := c.Request.Context().Value("csrf_token")
		if val != "mytoken" {
			t.Error(testutils.DiffMessage(val, "mytoken", "token must be stored in request context"))
		}
	})
}

// --- State-changing methods: valid token ---

func TestCSRF_POST_ValidHeader(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPost, csrfDefaultHeaderName, "tok", "tok")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "POST with valid header token must pass"))
	}
}

func TestCSRF_POST_ValidAltHeader(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPost, csrfAltHeader, "tok", "tok")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "POST with X-XSRF-TOKEN must pass"))
	}
}

func TestCSRF_POST_ValidFormField(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContext(http.MethodPost, "tok", "tok")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "POST with valid form field must pass"))
	}
}

func TestCSRF_PUT_ValidHeader(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPut, csrfDefaultHeaderName, "tok", "tok")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "PUT with valid token must pass"))
	}
}

func TestCSRF_PATCH_ValidHeader(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPatch, csrfDefaultHeaderName, "tok", "tok")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "PATCH with valid token must pass"))
	}
}

func TestCSRF_DELETE_ValidHeader(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodDelete, csrfDefaultHeaderName, "tok", "tok")
	called := false
	mw.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "DELETE with valid token must pass"))
	}
}

// --- State-changing methods: invalid / missing token ---

func TestCSRF_POST_MissingToken_Panics(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPost, csrfDefaultHeaderName, "", "tok")
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "missing token must panic"))
		}
	}()
	mw.Use(c, noop)
}

func TestCSRF_POST_WrongToken_Panics(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPost, csrfDefaultHeaderName, "wrong", "correct")
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "wrong token must panic"))
		}
	}()
	mw.Use(c, noop)
}

func TestCSRF_POST_SpecialCharsToken_Panics(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPost, csrfDefaultHeaderName, "../../../etc/passwd", "correct")
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "path traversal token must panic"))
		}
	}()
	mw.Use(c, noop)
}

func TestCSRF_POST_EmptyToken_Panics(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	c, _ := newCSRFContextWithHeader(http.MethodPost, csrfDefaultHeaderName, "", "tok")
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "empty submitted token must panic"))
		}
	}()
	mw.Use(c, noop)
}

// --- NewMiddleware pre-compilation ---

func TestCSRF_NewMiddleware_ReturnsCompiledCSRF(t *testing.T) {
	mw := CSRF{TokenLength: 16}.NewMiddleware()
	if _, ok := mw.(compiledCSRF); !ok {
		t.Error(testutils.DiffMessage(mw, "compiledCSRF", "NewMiddleware must return compiledCSRF"))
	}
}

// --- Concurrency ---

func TestCSRF_ConcurrentSafeRequests(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, _ := newCSRFContext(http.MethodGet, "", "")
			mw.Use(c, noop)
		}()
	}
	wg.Wait()
}

func TestCSRF_ConcurrentStateChanging(t *testing.T) {
	mw := CSRF{}.NewMiddleware()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, _ := newCSRFContextWithHeader(http.MethodPost, csrfDefaultHeaderName, "tok", "tok")
			mw.Use(c, noop)
		}()
	}
	wg.Wait()
}
