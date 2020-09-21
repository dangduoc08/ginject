package gogo

import (
	"net/http"
)

type routerGroup struct {
	routerMap   router
	middlewares []Handler // Router global middlewares
}

const (
	routeKey = iota
	handlersKey
)

// Router inits router group
// generate all route and handler map slice
func Router() Controller {
	var middlewares []Handler

	var gr *routerGroup = &routerGroup{
		routerMap:   newRouter(),
		middlewares: middlewares,
	}

	return gr
}

func (gr *routerGroup) Get(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.routerMap.insert(route, http.MethodGet, handlers...)
	return gr
}

func (gr *routerGroup) Post(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.routerMap.insert(route, http.MethodPost, handlers...)
	return gr
}

func (gr *routerGroup) Put(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.routerMap.insert(route, http.MethodPut, handlers...)
	return gr
}

func (gr *routerGroup) Delete(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.routerMap.insert(route, http.MethodDelete, handlers...)
	return gr
}

func (gr *routerGroup) Patch(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.routerMap.insert(route, http.MethodPatch, handlers...)
	return gr
}

func (gr *routerGroup) Head(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.routerMap.insert(route, http.MethodHead, handlers...)
	return gr
}

func (gr *routerGroup) Options(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.routerMap.insert(route, http.MethodOptions, handlers...)
	return gr
}

func (gr *routerGroup) Group(args ...interface{}) Controller {
	parentRoute, sourceRouterGroups := resolveRouterGroup(args...)
	parentRoute = handleSlash(parentRoute)
	mergeRouterGroup(gr, parentRoute, sourceRouterGroups)
	return gr
}

func (gr *routerGroup) Use(args ...interface{}) Controller {
	parentRoute, sourceHandlers := resolveMiddlewares(args...)
	parentRoute = handleSlash(parentRoute)
	mergeMiddleware(gr, parentRoute, sourceHandlers)
	return gr
}
