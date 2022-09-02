package routing

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/ds"
)

type routerData struct {
	Handlers *[]context.Handler
	Params   *context.Param[interface{}]
}

type Router struct {
	*ds.Trie
	array       []map[string]*routerData
	middlewares middleware
}

func NewRouter() *Router {
	trieInstance := ds.NewTrie()
	middlewareInstance := newMiddleware()

	return &Router{
		Trie:        trieInstance,
		array:       []map[string]*routerData{},
		middlewares: middlewareInstance,
	}
}

func (routerInstance *Router) add(route string, handlers ...context.Handler) *Router {
	routerAdapter := adapter{
		routerInstance,
	}
	routerAdapter.insert(route, handlers...)
	routerAdapter.serve(route, ADD)

	return routerInstance
}

func (routerInstance *Router) Match(route string) (bool, string, *routerData) {
	routerAdapter := adapter{
		routerInstance,
	}

	return routerAdapter.find(route)
}
