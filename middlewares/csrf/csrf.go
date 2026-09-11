package csrf

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

const (
	csrfDefaultTokenLength = 32
	csrfDefaultCookieName  = "_csrf"
	csrfDefaultHeaderName  = "X-CSRF-Token"
	csrfDefaultContextKey  = "csrf_token"
	csrfDefaultCookiePath  = "/"
	csrfAltHeader          = "X-XSRF-TOKEN"
	csrfFormField          = "_csrf"
)

var csrfSafeMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodHead:    true,
	http.MethodOptions: true,
}

type CSRF struct {
	TokenLength int    // default 32 bytes of entropy (64 hex chars)
	CookieName  string // default "_csrf"
	HeaderName  string // default "X-CSRF-Token"
	ContextKey  string // default "csrf_token"; token is stored in request context under this key
	CookiePath  string // default "/"

	// SameSite defaults to http.SameSiteLaxMode. Lax keeps the cookie off
	// cross-site POSTs while still surviving top-level navigation.
	SameSite http.SameSite

	// Secure forces the Secure attribute on. It is set automatically for
	// requests that arrived over TLS, so this is only needed behind a proxy
	// that terminates TLS upstream.
	Secure bool
}

type csrfOptions struct {
	tokenLength int
	cookieName  string
	headerName  string
	cookiePath  string
	contextKey  any
	sameSite    http.SameSite
	secure      bool
}

type compiledCSRF struct {
	opts *csrfOptions
}

func (instance CSRF) NewMiddleware() common.MiddlewareFn {
	return compiledCSRF{opts: loadCSRFOptions(&instance)}
}

func loadCSRFOptions(c *CSRF) *csrfOptions {
	opts := new(csrfOptions)

	opts.tokenLength = c.TokenLength
	if opts.tokenLength <= 0 {
		opts.tokenLength = csrfDefaultTokenLength
	}

	opts.cookieName = c.CookieName
	if opts.cookieName == "" {
		opts.cookieName = csrfDefaultCookieName
	}

	opts.headerName = c.HeaderName
	if opts.headerName == "" {
		opts.headerName = csrfDefaultHeaderName
	}

	opts.contextKey = c.ContextKey
	if opts.contextKey == "" {
		opts.contextKey = csrfDefaultContextKey
	}

	opts.cookiePath = c.CookiePath
	if opts.cookiePath == "" {
		opts.cookiePath = csrfDefaultCookiePath
	}

	// The zero value of http.SameSite is 0, which is not SameSiteDefaultMode (1)
	// and makes SetCookie omit the attribute entirely.
	opts.sameSite = c.SameSite
	if opts.sameSite == 0 {
		opts.sameSite = http.SameSiteLaxMode
	}

	opts.secure = c.Secure

	return opts
}

func GenerateCSRFToken(length int) (string, error) {
	if length <= 0 {
		length = csrfDefaultTokenLength
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func CompareTokensSecurely(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (m compiledCSRF) Use(r *http.Request, w http.ResponseWriter, next ctx.Next) {
	opts := m.opts

	var cookieToken string
	if cookie, err := r.Cookie(opts.cookieName); err == nil && cookie.Value != "" {
		cookieToken = cookie.Value
	} else {
		token, err := GenerateCSRFToken(opts.tokenLength)
		if err != nil {
			panic(exception.InternalServerErrorException("CSRF token generation failed"))
		}

		http.SetCookie(w, &http.Cookie{
			Name:     opts.cookieName,
			Value:    token,
			Path:     opts.cookiePath,
			HttpOnly: false,
			Secure:   opts.secure || r.TLS != nil,
			SameSite: opts.sameSite,
		})
		cookieToken = token
	}

	*r = *r.WithContext(
		context.WithValue(r.Context(), opts.contextKey, cookieToken),
	)

	if csrfSafeMethods[r.Method] {
		next()
		return
	}

	requestToken := r.Header.Get(opts.headerName)
	if requestToken == "" {
		requestToken = r.Header.Get(csrfAltHeader)
	}
	if requestToken == "" {
		requestToken = r.FormValue(csrfFormField)
	}

	if !CompareTokensSecurely(requestToken, cookieToken) {
		panic(exception.ForbiddenException("CSRF token invalid"))
	}

	next()
}

func (instance CSRF) Use(r *http.Request, w http.ResponseWriter, next ctx.Next) {
	compiledCSRF{opts: loadCSRFOptions(&instance)}.Use(r, w, next)
}
