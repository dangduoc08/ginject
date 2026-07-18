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
}

type csrfOptions struct {
	tokenLength int
	cookieName  string
	headerName  string
	contextKey  any
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

	return opts
}

// GenerateCSRFToken returns a hex-encoded cryptographically random token of
// the given byte length. The returned string is 2×length characters long.
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

// CompareTokensSecurely returns true if a and b are equal using a
// constant-time comparison to prevent timing attacks.
func CompareTokensSecurely(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (m compiledCSRF) Use(c *ctx.HTTPContext, next ctx.Next) {
	opts := m.opts

	// Retrieve or generate the token from the cookie.
	var cookieToken string
	if cookie, err := c.Cookie(opts.cookieName); err == nil && cookie.Value != "" {
		cookieToken = cookie.Value
	} else {
		token, err := GenerateCSRFToken(opts.tokenLength)
		if err != nil {
			panic(exception.InternalServerErrorException("CSRF token generation failed"))
		}
		// SameSite=Lax and Secure=true are recommended in production.
		// HttpOnly must be false so the JS client can read the token for
		// the double-submit cookie pattern.
		http.SetCookie(c.ResponseWriter, &http.Cookie{
			Name:     opts.cookieName,
			Value:    token,
			Path:     "/",
			HttpOnly: false,
		})
		cookieToken = token
	}

	// Expose the token to downstream handlers via request context.
	c.Request = c.WithContext(
		context.WithValue(c.Context(), opts.contextKey, cookieToken),
	)

	if csrfSafeMethods[c.Method] {
		next()
		return
	}

	// Extract the submitted token: header takes priority over form field.
	requestToken := c.Request.Header.Get(opts.headerName)
	if requestToken == "" {
		requestToken = c.Request.Header.Get(csrfAltHeader)
	}
	if requestToken == "" {
		requestToken = c.FormValue(csrfFormField)
	}

	if !CompareTokensSecurely(requestToken, cookieToken) {
		panic(exception.ForbiddenException("CSRF token invalid"))
	}

	next()
}

func (instance CSRF) Use(c *ctx.HTTPContext, next ctx.Next) {
	compiledCSRF{opts: loadCSRFOptions(&instance)}.Use(c, next)
}
