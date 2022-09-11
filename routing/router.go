package routing

import (
	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/ds"
)

type routerData struct {
	Handlers *[]ctx.Handler
	Params   *ctx.Param[interface{}]
}

type Router struct {
	*ds.Trie
	RouteMapDataArr []map[string]*routerData
	middlewares     middleware
}

func NewRouter() *Router {
	trieInstance := ds.NewTrie()
	middlewareInstance := newMiddleware()

	return &Router{
		Trie:            trieInstance,
		RouteMapDataArr: []map[string]*routerData{},
		middlewares:     middlewareInstance,
	}
}

func (routerInstance *Router) add(route string, handlers ...ctx.Handler) *Router {
	routerAdapter := adapter{
		routerInstance,
	}
	routerAdapter.insert(route, handlers...)
	routerAdapter.serve(route, ADD)

	return routerInstance
}

func (routerInstance *Router) match(route string) (bool, string, *routerData) {
	routerAdapter := adapter{
		routerInstance,
	}

	return routerAdapter.find(route)
}
