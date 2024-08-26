package middlewares

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/dangduoc08/gogo/ctx"
	"github.com/dangduoc08/gogo/utils"
)

type corsHeader map[string]string

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

func (h corsHeader) configureMaxAge(opts *CORSOptions) corsHeader {
	h["Access-Control-Max-Age"] = opts.maxAge // milliseconds
	return h
}

func (h corsHeader) configureAllowMethods(opts *CORSOptions) corsHeader {
	h["Access-Control-Allow-Methods"] = opts.allowMethods
	return h
}

func (h corsHeader) configureAllowOrigin(headers ctx.Header, opts *CORSOptions) corsHeader {
	requestOrigin := headers.Get("Origin")
	switch allowOrigin := opts.AllowOrigin.(type) {
	case string:
		h["Access-Control-Allow-Origin"] = allowOrigin
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

func (h corsHeader) configureAllowHeaders(headers ctx.Header, opts *CORSOptions) corsHeader {
	if opts.allowHeaders != "" {
		h["Access-Control-Allow-Headers"] = opts.allowHeaders
	} else {
		allowedHeaders := headers.Get("access-control-request-headers")
		h.vary("Access-Control-Request-Headers")
		h["Access-Control-Allow-Headers"] = allowedHeaders
	}

	return h
}

func (h corsHeader) configureExposeHeaders(opts *CORSOptions) corsHeader {
	if opts.exposeHeaders != "" {
		h["Access-Control-Expose-Headers"] = opts.exposeHeaders
	}

	return h
}

func (h corsHeader) configureAllowCredentials(opts *CORSOptions) corsHeader {
	if opts.IsAllowCredentials {
		h["Access-Control-Allow-Credentials"] = "true"
	}

	return h
}

func (h corsHeader) applyHeaders(c *ctx.Context) {
	for headerKey, headerValue := range h {
		c.ResponseWriter.Header().Set(headerKey, headerValue)
	}
}

type CORSOptions struct {
	AllowOrigin any // string | []string | regexp

	AllowHeaders any    // string | []string
	allowHeaders string // string | []string

	ExposeHeaders any // string | []string
	exposeHeaders string

	AllowMethods []string
	allowMethods string

	MaxAge int
	maxAge string

	IsAllowCredentials   bool
	IsPreflightContinue  bool
	OptionsSuccessStatus int
	// /**
	//  * @default false
	//  */
	// preflightContinue? boolean | undefined;
	// /**
	//  * @default 204
	//  */
	// optionsSuccessStatus? number | undefined;
}

func loadCORSOptions(opts *CORSOptions) *CORSOptions {
	if opts == nil {
		opts = &CORSOptions{}
	}

	if opts.OptionsSuccessStatus == 0 {
		opts.OptionsSuccessStatus = 204
	}

	if opts.MaxAge == 0 {

		// default Access-Control-Max-Age = 5 seconds
		opts.MaxAge = 5000
	}
	opts.maxAge = strconv.Itoa(opts.MaxAge / 1000) // milliseconds

	if len(opts.AllowMethods) == 0 {
		opts.AllowMethods = []string{
			http.MethodGet,
			http.MethodHead,
			http.MethodPut,
			http.MethodPatch,
			http.MethodPost,
			http.MethodDelete,
		}
	}
	opts.allowMethods = strings.Join(opts.AllowMethods, ", ")

	if opts.AllowOrigin == nil {
		opts.AllowOrigin = "*"
	} else if allowOrigins, ok := opts.AllowOrigin.([]string); ok {
		m := map[string]bool{}
		for _, allowOrigin := range allowOrigins {
			m[utils.StrRemoveEnd(allowOrigin, "/")] = true
		}
		opts.AllowOrigin = m
	}

	if allowHeaders, ok := opts.AllowHeaders.([]string); ok {
		opts.allowHeaders = strings.Join(allowHeaders, ", ")
	}

	if exposeHeaders, ok := opts.ExposeHeaders.([]string); ok {
		opts.exposeHeaders = strings.Join(exposeHeaders, ", ")
	}

	// origin: '*',
	// methods: 'GET,HEAD,PUT,PATCH,POST,DELETE',
	// preflightContinue: false,
	// optionsSuccessStatus: 204

	return opts
}

func CORS(opts *CORSOptions) func(*ctx.Context) {
	opts = loadCORSOptions(opts)

	return func(c *ctx.Context) {
		requestHeaders := c.Header()
		h := corsHeader{}

		h.
			configureMaxAge(opts).
			configureAllowMethods(opts).
			configureAllowOrigin(requestHeaders, opts).
			configureAllowHeaders(requestHeaders, opts).
			configureExposeHeaders(opts).
			configureAllowCredentials(opts).
			applyHeaders(c)

		// methods which not in GET, POST, HEAD
		// Content-Type not in 'application/x-www-form-urlencoded', 'multipart/form-data', 'text/plain'
		// request include credentials
		// will have preflight request
		if c.Method == http.MethodOptions {
			if opts.IsPreflightContinue {
				c.Next()
			} else {

				// Safari (and potentially other browsers) need content-length 0,
				// for 204 or they just hang waiting for a body
				// c.Status(opts.OptionsSuccessStatus)
				c.Status(opts.OptionsSuccessStatus)
				c.ResponseWriter.WriteHeader(opts.OptionsSuccessStatus)
				c.ResponseWriter.Header().Set("Content-Length", "0")
				c.Event.Emit(ctx.REQUEST_FINISHED, c)
			}

			return
		}

		c.Next()
	}
}
