package helmet

import (
	"net/http"
	"strconv"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

const helmetDefaultCSP = "default-src 'self';base-uri 'self';font-src 'self' https: data:;form-action 'self';frame-ancestors 'self';img-src 'self' data:;object-src 'none';script-src 'self';script-src-attr 'none';style-src 'self' https: 'unsafe-inline';upgrade-insecure-requests"

type Helmet struct {
	ContentSecurityPolicy        string // "" = helmetDefaultCSP
	CrossOriginEmbedderPolicy    string // "" = "require-corp"
	CrossOriginOpenerPolicy      string // "" = "same-origin"
	CrossOriginResourcePolicy    string // "" = "same-origin"
	DNSPrefetchControl           string // "" = "off"
	FrameOptions                 string // "" = "SAMEORIGIN"
	HSTSMaxAge                   int    // 0 = 15552000 (180 days)
	HSTSExcludeSubDomains        bool   // false = include subdomains
	HSTSPreload                  bool
	DisableHSTS                  bool
	PermittedCrossDomainPolicies string // "" = "none"
	ReferrerPolicy               string // "" = "no-referrer"
}

type helmetOptions struct {
	contentSecurityPolicy        string
	crossOriginEmbedderPolicy    string
	crossOriginOpenerPolicy      string
	crossOriginResourcePolicy    string
	dnsPrefetchControl           string
	frameOptions                 string
	hsts                         string
	permittedCrossDomainPolicies string
	referrerPolicy               string
}

type compiledHelmet struct {
	opts *helmetOptions
}

func (instance Helmet) NewMiddleware() common.MiddlewareFn {
	return compiledHelmet{opts: loadHelmetOptions(&instance)}
}

func loadHelmetOptions(h *Helmet) *helmetOptions {
	opts := new(helmetOptions)

	opts.contentSecurityPolicy = h.ContentSecurityPolicy
	if opts.contentSecurityPolicy == "" {
		opts.contentSecurityPolicy = helmetDefaultCSP
	}

	opts.crossOriginEmbedderPolicy = h.CrossOriginEmbedderPolicy
	if opts.crossOriginEmbedderPolicy == "" {
		opts.crossOriginEmbedderPolicy = "require-corp"
	}

	opts.crossOriginOpenerPolicy = h.CrossOriginOpenerPolicy
	if opts.crossOriginOpenerPolicy == "" {
		opts.crossOriginOpenerPolicy = "same-origin"
	}

	opts.crossOriginResourcePolicy = h.CrossOriginResourcePolicy
	if opts.crossOriginResourcePolicy == "" {
		opts.crossOriginResourcePolicy = "same-origin"
	}

	opts.dnsPrefetchControl = h.DNSPrefetchControl
	if opts.dnsPrefetchControl == "" {
		opts.dnsPrefetchControl = "off"
	}

	opts.frameOptions = h.FrameOptions
	if opts.frameOptions == "" {
		opts.frameOptions = "SAMEORIGIN"
	}

	opts.permittedCrossDomainPolicies = h.PermittedCrossDomainPolicies
	if opts.permittedCrossDomainPolicies == "" {
		opts.permittedCrossDomainPolicies = "none"
	}

	opts.referrerPolicy = h.ReferrerPolicy
	if opts.referrerPolicy == "" {
		opts.referrerPolicy = "no-referrer"
	}

	if !h.DisableHSTS {
		maxAge := h.HSTSMaxAge
		if maxAge == 0 {
			maxAge = 15552000
		}
		hsts := "max-age=" + strconv.Itoa(maxAge)
		if !h.HSTSExcludeSubDomains {
			hsts += "; includeSubDomains"
		}
		if h.HSTSPreload {
			hsts += "; preload"
		}
		opts.hsts = hsts
	}

	return opts
}

func (m compiledHelmet) Use(r *http.Request, w http.ResponseWriter, next ctx.Next) {
	opts := m.opts

	rh := w.Header()
	rh.Set("Content-Security-Policy", opts.contentSecurityPolicy)
	rh.Set("Cross-Origin-Embedder-Policy", opts.crossOriginEmbedderPolicy)
	rh.Set("Cross-Origin-Opener-Policy", opts.crossOriginOpenerPolicy)
	rh.Set("Cross-Origin-Resource-Policy", opts.crossOriginResourcePolicy)
	rh.Set("X-Content-Type-Options", "nosniff")
	rh.Set("X-DNS-Prefetch-Control", opts.dnsPrefetchControl)
	rh.Set("X-Download-Options", "noopen")
	rh.Set("X-Frame-Options", opts.frameOptions)
	rh.Set("X-Permitted-Cross-Domain-Policies", opts.permittedCrossDomainPolicies)
	rh.Set("X-XSS-Protection", "0")
	rh.Set("Origin-Agent-Cluster", "?1")
	rh.Set("Referrer-Policy", opts.referrerPolicy)

	if opts.hsts != "" {
		rh.Set("Strict-Transport-Security", opts.hsts)
	}

	next()
}

func (instance Helmet) Use(r *http.Request, w http.ResponseWriter, next ctx.Next) {
	compiledHelmet{opts: loadHelmetOptions(&instance)}.Use(r, w, next)
}
