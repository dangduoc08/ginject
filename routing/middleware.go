package routing

import "github.com/dangduoc08/gooh/context"

type middleware []map[string]*[]context.Handler

func newMiddleware() middleware {
	return make([]map[string]*([]context.Handler), 0)
}

func (middlewareInstance *middleware) cache(route string, handlers ...context.Handler) {
	middlewareMapRouteToHandler := map[string]*[]context.Handler{
		route: &handlers,
	}

	*middlewareInstance = append(*middlewareInstance, middlewareMapRouteToHandler)
}
