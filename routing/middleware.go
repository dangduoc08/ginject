package routing

import "github.com/dangduoc08/gooh/ctx"

type middleware []map[string]*[]ctx.Handler

func newMiddleware() middleware {
	return make([]map[string]*([]ctx.Handler), 0)
}

func (middlewareInstance *middleware) cache(route string, handlers ...ctx.Handler) {
	middlewareMapRouteToHandler := map[string]*[]ctx.Handler{
		route: &handlers,
	}

	*middlewareInstance = append(*middlewareInstance, middlewareMapRouteToHandler)
}
