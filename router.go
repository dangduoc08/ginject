package gogo

import (
	"net/http"
)

// router map holds struct:
//	{
//		'GET': [ <routeAndHandlerMapSlice>
//			{ <routeAndHandlerMap>
//				route: '/get'
//				handlers: [
//					...? router global middlwares
//					1st_Handler,
//					2nd_handler,
//					3rd_handler
//					...
//				]
//			}
//		]
//		...
//		'DELETE': [ ... ]
//	}
//
// **********************Example**********************
//
// 	routerGroup := {
//		"GET": [
//			{
//				route: "/test"
//				handlers: [
//					func add(a, b int) int { return a + b },
//					func minus(a, b int) int { return a - b }
//				]
//			}
//		],
//		"POST": [
//			{
//				route: "/test"
//				handlers: [
//					func mul(a, b int) int { return a * b },
//					func div(a, b int) int { return a / b }
//				]
//			}
//		],
//	}
type router map[string][]map[int]interface{}

type routerGroup struct {
	router      router
	middlewares []Handler // Router global middlewares
}

const (
	routeKey = iota
	handlersKey
)

// Router inits router group
// generate all route and handler map slice
func Router() Controller {
	router := make(map[string][]map[int]interface{})
	var routeAndHandlerMapSlice []map[int]interface{}
	var middlewares []Handler

	// Append empty route and handler map slice
	// into router
	for _, httpMethod := range httpMethods {
		router[httpMethod] = routeAndHandlerMapSlice
	}

	var gr *routerGroup = &routerGroup{
		router:      router,
		middlewares: middlewares,
	}

	return gr
}

// Check route whethere existed in router group
// return isMatch and position
// in route and handler map slice
func (gr *routerGroup) match(route, httpMethod string) (bool, int) {
	router := gr.router
	routeAndHandlerMapSlice := router[httpMethod]
	var isMatch bool
	var position int = -1

	for index, routeAndHandlerMap := range routeAndHandlerMapSlice {
		if route == routeAndHandlerMap[routeKey] {
			isMatch = true
			position = index
			break
		}
	}

	return isMatch, position
}

// Push routes, handlers to router group
// if route was inited before, we will append handler
// else append new route and handler map
func (gr *routerGroup) insert(route, httpMethod string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Nil handler")
	}
	router := gr.router
	routeAndHandlerMapSlice := router[httpMethod]
	isExistedInRouterGroup, position := gr.match(route, httpMethod)

	if isExistedInRouterGroup {

		// Route was inserted before
		// therefore just need appending handlers
		routeAndHandlerMapSlice[position][handlersKey] = append(
			routeAndHandlerMapSlice[position][handlersKey].([]Handler),
			handlers...,
		)
	} else {

		// Init new map
		// and append new route and handler map
		newRouteAndHandlerMap := make(map[int]interface{})
		newRouteAndHandlerMap[routeKey] = route
		newRouteAndHandlerMap[handlersKey] = handlers
		router[httpMethod] = append(router[httpMethod], newRouteAndHandlerMap)
	}
}

func (gr *routerGroup) Get(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.insert(route, http.MethodGet, handlers...)
	return gr
}

func (gr *routerGroup) Post(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.insert(route, http.MethodPost, handlers...)
	return gr
}

func (gr *routerGroup) Put(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.insert(route, http.MethodPut, handlers...)
	return gr
}

func (gr *routerGroup) Delete(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.insert(route, http.MethodDelete, handlers...)
	return gr
}

func (gr *routerGroup) Patch(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.insert(route, http.MethodPatch, handlers...)
	return gr
}

func (gr *routerGroup) Head(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.insert(route, http.MethodHead, handlers...)
	return gr
}

func (gr *routerGroup) Options(route string, handlers ...Handler) Controller {
	route = handleSlash(route)
	gr.insert(route, http.MethodOptions, handlers...)
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
