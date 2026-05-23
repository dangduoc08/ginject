package httpclient

import "net/http"

// Handler executes an HTTP request and returns a Response.
type Handler func(*http.Request) (*Response, error)

// Middleware wraps a Handler, adding pre/post processing.
type Middleware func(Handler) Handler

func buildChain(middlewares []Middleware, final Handler) Handler {
	h := final
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
