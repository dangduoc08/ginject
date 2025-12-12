package middlewares

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/utils"
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
	// /**
	//  * @default false
	//  */
	// preflightContinue? boolean | undefined;
	// /**
	//  * @default 204
	//  */
	// optionsSuccessStatus? number | undefined;
}

type corsOptions struct {
	optionsSuccessStatus int
	isAllowCredentials   bool
	allowOrigin          any    // string | []string | regexp
	allowHeaders         string // string | []string
	exposeHeaders        string
	allowMethods         string
	maxAge               string
}

func (instance CORS) NewMiddleware() common.MiddlewareFn {

	return instance
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
	h["Access-Control-Max-Age"] = opts.maxAge // milliseconds
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

	opts.allowOrigin = cors.AllowOrigin
	opts.isAllowCredentials = cors.IsAllowCredentials

	if cors.OptionsSuccessStatus == 0 {
		opts.optionsSuccessStatus = 204
	} else {
		opts.optionsSuccessStatus = cors.OptionsSuccessStatus
	}

	if cors.MaxAge == 0 {

		// default Access-Control-Max-Age = 5 seconds
		cors.MaxAge = 5000
	}
	opts.maxAge = strconv.Itoa(cors.MaxAge / 1000) // milliseconds

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
		m := map[string]bool{}
		for _, allowOrigin := range allowOrigins {
			m[utils.StrRemoveEnd(allowOrigin, "/")] = true
		}
		cors.AllowOrigin = m
	}

	if allowHeaders, ok := cors.AllowHeaders.([]string); ok {
		opts.allowHeaders = strings.Join(allowHeaders, ", ")
	}

	if exposeHeaders, ok := cors.ExposeHeaders.([]string); ok {
		opts.exposeHeaders = strings.Join(exposeHeaders, ", ")
	}

	// origin: '*',
	// methods: 'GET,HEAD,PUT,PATCH,POST,DELETE',
	// preflightContinue: false,
	// optionsSuccessStatus: 204

	return opts
}

func (instance CORS) Use(c *ctx.Context, next ctx.Next) {
	opts := loadCORSOptions(&instance)

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
		if instance.IsPreflightContinue {
			c.Next()
		} else {

			// Safari (and potentially other browsers) need content-length 0,
			// for 204 or they just hang waiting for a body
			// c.Status(opts.OptionsSuccessStatus)
			c.Status(opts.optionsSuccessStatus)
			c.ResponseWriter.WriteHeader(opts.optionsSuccessStatus)
			c.ResponseWriter.Header().Set("Content-Length", "0")
			c.Event.Emit(ctx.REQUEST_FINISHED, c)
		}

		return
	}

	next()
}
