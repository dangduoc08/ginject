package cors

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

const defaultMaxAge = 5 * time.Second

var defaultAllowMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPut,
	http.MethodPatch,
	http.MethodPost,
	http.MethodDelete,
}

type CORS struct {
	AllowOrigin any // string | []string | *regexp.Regexp

	AllowHeaders any // string | []string

	ExposeHeaders any // string | []string

	AllowMethods []string

	MaxAge time.Duration

	IsAllowCredentials   bool
	IsPreflightContinue  bool
	OptionsSuccessStatus int
}

// allowOrigin is always one of: "*" | map[string]bool | *regexp.Regexp,
// regardless of what shape the user configured it in.
type corsOptions struct {
	optionsSuccessStatus int
	isAllowCredentials   bool
	isPreflightContinue  bool
	shouldVaryOrigin     bool
	allowOrigin          any
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

func normalizeOrigin(origin string) string {
	return strings.TrimSuffix(origin, "/")
}

func normalizeAllowOrigin(allowOrigin any) any {
	switch v := allowOrigin.(type) {
	case nil:
		return "*"
	case string:
		if v == "*" {
			return "*"
		}
		return map[string]bool{normalizeOrigin(v): true}
	case []string:
		m := make(map[string]bool, len(v))
		for _, origin := range v {
			m[normalizeOrigin(origin)] = true
		}
		return m
	case *regexp.Regexp:
		return v
	default:
		return map[string]bool{}
	}
}

func shouldVaryOrigin(allowOrigin any, allowCredentials bool) bool {
	_, isWildcard := allowOrigin.(string)
	return !isWildcard || allowCredentials
}

func matchOrigin(allowOrigin any, requestOrigin string, allowCredentials bool) (string, bool) {
	switch ao := allowOrigin.(type) {
	case string: // always "*" after normalization
		if allowCredentials {
			if requestOrigin == "" || requestOrigin == "null" {
				return "", false
			}
			return requestOrigin, true
		}
		return "*", true
	case map[string]bool:
		if ao[normalizeOrigin(requestOrigin)] {
			return requestOrigin, true
		}
		return "", false
	case *regexp.Regexp:
		if ao.MatchString(normalizeOrigin(requestOrigin)) {
			return requestOrigin, true
		}
		return "", false
	default:
		return "", false
	}
}

func appendVary(vary, token string) string {
	if token == "" {
		return vary
	}
	if vary == "" {
		return token
	}
	return vary + ", " + token
}

// mergeVary adds addition's tokens into any Vary header already present,
// case-insensitively deduped, instead of overwriting it.
func mergeVary(header http.Header, addition string) {
	if addition == "" {
		return
	}

	existing := header.Values("Vary")
	if len(existing) == 0 {
		header.Set("Vary", addition)
		return
	}

	seen := make(map[string]struct{}, len(existing)+2)
	merged := make([]string, 0, len(existing)+2)
	collect := func(list string) {
		for _, tok := range strings.Split(list, ",") {
			tok = strings.TrimSpace(tok)
			if tok == "" {
				continue
			}
			key := strings.ToLower(tok)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, tok)
		}
	}

	for _, v := range existing {
		collect(v)
	}
	collect(addition)

	header.Set("Vary", strings.Join(merged, ", "))
}

func configureAllowOrigin(header http.Header, requestOrigin string, allowOrigin any, allowCredentials, shouldVaryOrigin bool) bool {
	if value, matched := matchOrigin(allowOrigin, requestOrigin, allowCredentials); matched {
		header.Set("Access-Control-Allow-Origin", value)
	}
	return shouldVaryOrigin
}

func configureAllowHeaders(header http.Header, requestHeaders ctx.Header, allowHeaders string) bool {
	if allowHeaders != "" {
		header.Set("Access-Control-Allow-Headers", allowHeaders)
		return false
	}

	header.Set("Access-Control-Allow-Headers", requestHeaders.Get("Access-Control-Request-Headers"))
	return true
}

func loadCORSOptions(cors *CORS) *corsOptions {
	opts := &corsOptions{
		isAllowCredentials:   cors.IsAllowCredentials,
		isPreflightContinue:  cors.IsPreflightContinue,
		optionsSuccessStatus: cors.OptionsSuccessStatus,
	}

	if opts.optionsSuccessStatus == 0 {
		opts.optionsSuccessStatus = http.StatusNoContent
	}

	maxAge := cors.MaxAge
	if maxAge <= 0 {
		maxAge = defaultMaxAge
	}
	opts.maxAge = strconv.Itoa(int(maxAge / time.Second))

	allowMethods := cors.AllowMethods
	if len(allowMethods) == 0 {
		allowMethods = defaultAllowMethods
	}
	opts.allowMethods = strings.Join(allowMethods, ", ")

	opts.allowOrigin = normalizeAllowOrigin(cors.AllowOrigin)
	opts.shouldVaryOrigin = shouldVaryOrigin(opts.allowOrigin, opts.isAllowCredentials)

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

func (m compiledCORS) Use(c *ctx.HTTPContext, next ctx.Next) {
	opts := m.opts

	requestHeaders := c.Header()
	requestOrigin := requestHeaders.Get("Origin")
	if requestOrigin == "" {
		next()
		return
	}

	if c.GetType() == ctx.WSType {
		if _, matched := matchOrigin(opts.allowOrigin, requestOrigin, opts.isAllowCredentials); matched {
			next()
		}
		return
	}

	header := c.ResponseWriter.Header()

	var vary string
	if configureAllowOrigin(header, requestOrigin, opts.allowOrigin, opts.isAllowCredentials, opts.shouldVaryOrigin) {
		vary = appendVary(vary, "Origin")
	}
	if opts.exposeHeaders != "" {
		header.Set("Access-Control-Expose-Headers", opts.exposeHeaders)
	}
	if opts.isAllowCredentials {
		header.Set("Access-Control-Allow-Credentials", "true")
	}

	isPreflight := c.Method == http.MethodOptions
	if isPreflight {
		header.Set("Access-Control-Max-Age", opts.maxAge)
		header.Set("Access-Control-Allow-Methods", opts.allowMethods)
		if configureAllowHeaders(header, requestHeaders, opts.allowHeaders) {
			vary = appendVary(vary, "Access-Control-Request-Headers")
		}
	}

	mergeVary(header, vary)

	if isPreflight {
		if opts.isPreflightContinue {
			c.Next()
		} else {
			c.Status(opts.optionsSuccessStatus)
			header.Set("Content-Length", "0")
			c.WriteHeader(opts.optionsSuccessStatus)
			_ = c.Broker.Publish(ctx.RequestFinished, c)
		}

		return
	}

	next()
}

func (instance CORS) Use(c *ctx.HTTPContext, next ctx.Next) {
	compiledCORS{opts: loadCORSOptions(&instance)}.Use(c, next)
}
