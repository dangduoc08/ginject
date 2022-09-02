package routing

import (
	"github.com/dangduoc08/gooh/core"
)

type middleware []map[string]*[]core.Handler

func newMiddleware() middleware {
	return make([]map[string]*([]core.Handler), 0)
}

func (middlewareInstance *middleware) cache(route string, handlers ...core.Handler) {
	middlewareMapRouteToHandler := map[string]*[]core.Handler{
		route: &handlers,
	}

	*middlewareInstance = append(*middlewareInstance, middlewareMapRouteToHandler)
}
