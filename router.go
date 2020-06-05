package gogo

import (
	"net/http"
)

// router map holds struct:
//	{
//		'GET': [ <routeAndHandlerArray>
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
type router map[string][]map[string]interface{}

type routerGroup struct {
	router      router
	middlewares []Handler // router global middlewares
}

var httpMethods []string = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodHead,
	http.MethodOptions,
	http.MethodDelete,
}

const (
	routeKey    = "route"
	handlersKey = "handlers"
)

// Router inits router map
// includes method arrays
func Router() Controller {
	router := make(map[string][]map[string]interface{})
	var middlewares []Handler
	routeAndHandlerArray := []map[string]interface{}{}

	for _, httpMethod := range httpMethods {
		router[httpMethod] = routeAndHandlerArray
	}

	var r routerGroup = routerGroup{
		router:      router,
		middlewares: middlewares,
	}

	return &r
}

// Iterable each router
func (r *router) forEach(callback func(httpMethod string, routeAndHandlerMap map[string]interface{})) {
	for httpMethod := range *r {
		routeAndHandlerArray := (*r)[httpMethod]
		if len(routeAndHandlerArray) > 0 {
			for _, routeAndHandlerMap := range routeAndHandlerArray {
				callback(httpMethod, routeAndHandlerMap)
			}
		}
	}
}

// Push routes, handlers to router group
// if route was inited before, we will append handler
func (gr *routerGroup) insert(route, httpMethod string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Nil handler")
	}
	router := gr.router
	routeAndHandlerArray := router[httpMethod]
	var hasRouteInsertedBefore bool
	var tmpRouteAndHandlerMap map[string]interface{}

	// Check whether route has inserted before ?
	for _, routeAndHandlerMap := range routeAndHandlerArray {
		if route == routeAndHandlerMap[routeKey] {
			hasRouteInsertedBefore = true
			tmpRouteAndHandlerMap = routeAndHandlerMap
			break
		}
	}

	if hasRouteInsertedBefore {

		// Route was inserted before
		// therefore just need appending handlers
		tmpRouteAndHandlerMap[handlersKey] = append(
			tmpRouteAndHandlerMap[handlersKey].([]func(*Request, ResponseExtender, func())),
			handlers...,
		)
	} else {

		// Init new map
		// and append new route and handler map
		tmpRouteAndHandlerMap = make(map[string]interface{})
		tmpRouteAndHandlerMap[routeKey] = route
		tmpRouteAndHandlerMap[handlersKey] = handlers
		router[httpMethod] = append(router[httpMethod], tmpRouteAndHandlerMap)
	}
}

func (gr *routerGroup) Get(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodGet, handlers...)
	return gr
}

func (gr *routerGroup) Post(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodPost, handlers...)
	return gr
}

func (gr *routerGroup) Put(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodPut, handlers...)
	return gr
}

func (gr *routerGroup) Patch(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodPatch, handlers...)
	return gr
}

func (gr *routerGroup) Head(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodHead, handlers...)
	return gr
}

func (gr *routerGroup) Options(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodOptions, handlers...)
	return gr
}

func (gr *routerGroup) Delete(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodDelete, handlers...)
	return gr
}

func (gr *routerGroup) UseRouter(args ...interface{}) Controller {
	parentRoute, sourceRouterGroups := resolveRouterGroup(args...)
	parentRoute = formatRoute(parentRoute)
	mergeRouterWithRouter(parentRoute, gr, sourceRouterGroups)
	return gr
}

func (gr *routerGroup) UseMiddleware(args ...interface{}) Controller {
	parentRoute, sourceHandlers := resolveMiddlewares(args...)
	parentRoute = formatRoute(parentRoute)

	// To check whether route of HTTP method exists or not
	// if exist this HTTP method will set to map
	// if not, will create all router http method routers
	var tmpInitializedHTTPMethods = make(map[string]bool)

	if parentRoute != "" {

		// Middlewares will add to each router
		for httpMethod, routeAndHandlerArray := range gr.router {
			for _, routeAndHandlerMap := range routeAndHandlerArray {
				var route string = routeAndHandlerMap[routeKey].(string)

				// Router has inited this route
				// therefore middlewares will add to its handler
				if route == parentRoute {
					var targetHandlers []Handler = routeAndHandlerMap[handlersKey].([]Handler)
					routeAndHandlerMap[handlersKey] = append(sourceHandlers, targetHandlers...)
					tmpInitializedHTTPMethods[httpMethod] = true
				}
			}
		}

		// Create remain http methods router
		for _, httpMethod := range httpMethods {
			if !tmpInitializedHTTPMethods[httpMethod] {
				gr.insert(parentRoute, httpMethod, sourceHandlers...)
			}
		}
	} else {

		// Middlewares will add to router global middlewares
		gr.middlewares = append(gr.middlewares, sourceHandlers...)
	}
	return gr
}
