package httpclient

import "net/http"

type Handler func(*http.Request) (*Response, error)

type Middleware func(Handler) Handler

func buildChain(middlewares []Middleware, final Handler) Handler {
	h := final
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
