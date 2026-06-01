package middlewares

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

type corsHeader map[string]string

type CORS struct {
	AllowOrigin any // string | []string | regexp

	AllowHeaders any // string | []string

	ExposeHeaders any // string | []string

	AllowMethods []string

	MaxAge int

	IsAllowCredentials   bool
	IsPreflightContinue  bool
	OptionsSuccessStatus int
}

type corsOptions struct {
	optionsSuccessStatus int
	isAllowCredentials   bool
	isPreflightContinue  bool
	allowOrigin          any // string | map[string]bool | *regexp.Regexp
	allowHeaders         string
	exposeHeaders        string
	allowMethods         string
	maxAge               string
}

type compiledCORS struct {
	opts *corsOptions
}

func (instance CORS) NewMiddleware() common.MiddlewareFn {
	return compiledCORS{opts: loadCORSOptions(&instance)}
}

func (h corsHeader) vary(k string) {
	if k == "" {
		return
	}

	if h["Vary"] == "" {
		h["Vary"] = k
		return
	}

	h["Vary"] = h["Vary"] + ", " + k
}

func (h corsHeader) configureMaxAge(opts *corsOptions) corsHeader {
	h["Access-Control-Max-Age"] = opts.maxAge
	return h
}

func (h corsHeader) configureAllowMethods(opts *corsOptions) corsHeader {
	h["Access-Control-Allow-Methods"] = opts.allowMethods
	return h
}

func (h corsHeader) configureAllowOrigin(headers ctx.Header, opts *corsOptions) corsHeader {
	requestOrigin := headers.Get("Origin")
	switch allowOrigin := opts.allowOrigin.(type) {
	case string:
		if allowOrigin == "*" && opts.isAllowCredentials {
			if requestOrigin != "null" {
				h["Access-Control-Allow-Origin"] = requestOrigin
				h.vary("Origin")
			}
		} else {
			h["Access-Control-Allow-Origin"] = allowOrigin
			if allowOrigin != "*" {
				h.vary("Origin")
			}
		}
		return h

	case map[string]bool:
		if _, ok := allowOrigin[requestOrigin]; ok {
			h["Access-Control-Allow-Origin"] = requestOrigin
			h.vary("Origin")
		}
		return h

	case *regexp.Regexp:
		if allowOrigin.MatchString(requestOrigin) {
			h["Access-Control-Allow-Origin"] = requestOrigin
			h.vary("Origin")
		}
		return h

	default:
		return h
	}
}

func (h corsHeader) configureAllowHeaders(headers ctx.Header, opts *corsOptions) corsHeader {
	if opts.allowHeaders != "" {
		h["Access-Control-Allow-Headers"] = opts.allowHeaders
	} else {
		allowedHeaders := headers.Get("access-control-request-headers")
		h.vary("Access-Control-Request-Headers")
		h["Access-Control-Allow-Headers"] = allowedHeaders
	}

	return h
}

func (h corsHeader) configureExposeHeaders(opts *corsOptions) corsHeader {
	if opts.exposeHeaders != "" {
		h["Access-Control-Expose-Headers"] = opts.exposeHeaders
	}

	return h
}

func (h corsHeader) configureAllowCredentials(opts *corsOptions) corsHeader {
	if opts.isAllowCredentials {
		h["Access-Control-Allow-Credentials"] = "true"
	}

	return h
}

func (h corsHeader) applyHeaders(c *ctx.Context) {
	for headerKey, headerValue := range h {
		c.ResponseWriter.Header().Set(headerKey, headerValue)
	}
}

func loadCORSOptions(cors *CORS) *corsOptions {
	opts := new(corsOptions)

	opts.isAllowCredentials = cors.IsAllowCredentials
	opts.isPreflightContinue = cors.IsPreflightContinue

	if cors.OptionsSuccessStatus == 0 {
		opts.optionsSuccessStatus = 204
	} else {
		opts.optionsSuccessStatus = cors.OptionsSuccessStatus
	}

	if cors.MaxAge == 0 {
		cors.MaxAge = 5000
	}
	opts.maxAge = strconv.Itoa(cors.MaxAge / 1000)

	if len(cors.AllowMethods) == 0 {
		cors.AllowMethods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
		}
	}
	opts.allowMethods = strings.Join(cors.AllowMethods, ", ")

	if cors.AllowOrigin == nil {
		cors.AllowOrigin = "*"
	} else if allowOrigins, ok := cors.AllowOrigin.([]string); ok {
		m := make(map[string]bool, len(allowOrigins))
		for _, allowOrigin := range allowOrigins {
			m[strings.TrimSuffix(allowOrigin, "/")] = true
		}
		cors.AllowOrigin = m
	}
	opts.allowOrigin = cors.AllowOrigin

	switch v := cors.AllowHeaders.(type) {
	case string:
		opts.allowHeaders = v
	case []string:
		opts.allowHeaders = strings.Join(v, ", ")
	}

	switch v := cors.ExposeHeaders.(type) {
	case string:
		opts.exposeHeaders = v
	case []string:
		opts.exposeHeaders = strings.Join(v, ", ")
	}

	return opts
}

func (m compiledCORS) Use(c *ctx.Context, next ctx.Next) {
	opts := m.opts

	requestHeaders := c.Header()
	if requestHeaders.Get("Origin") == "" {
		next()
		return
	}

	h := corsHeader{}
	h.configureAllowOrigin(requestHeaders, opts)
	h.configureExposeHeaders(opts)
	h.configureAllowCredentials(opts)

	if c.Method == http.MethodOptions {
		h.configureMaxAge(opts)
		h.configureAllowMethods(opts)
		h.configureAllowHeaders(requestHeaders, opts)
	}

	h.applyHeaders(c)

	if c.Method == http.MethodOptions {
		if opts.isPreflightContinue {
			c.Next()
		} else {
			c.Status(opts.optionsSuccessStatus)
			c.WriteHeader(opts.optionsSuccessStatus)
			c.ResponseWriter.Header().Set("Content-Length", "0")
			_ = c.Broker.Publish(ctx.REQUEST_FINISHED, c)
		}

		return
	}

	next()
}

func (instance CORS) Use(c *ctx.Context, next ctx.Next) {
	compiledCORS{opts: loadCORSOptions(&instance)}.Use(c, next)
}
