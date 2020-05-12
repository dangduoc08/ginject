package gogo

import (
	"fmt"
	"net/http"
)

// router map holds struct:
//	{
//		'GET': [ <routeAndHandlerArray>
//			{ <routeAndHandlerMap>
//				route: '/get'
//				handlers: [
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
type router map[string][]map[string]interface{}

type routerGroup struct {
	router      router
	middlewares []Handler // global middlewares
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
func (gr *routerGroup) insert(route, httpMethod string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Nil handler")
	}
	router := gr.router
	routeAndHandlerMap := make(map[string]interface{})
	routeAndHandlerMap[routeKey] = route
	routeAndHandlerMap[handlersKey] = handlers
	router[httpMethod] = append(router[httpMethod], routeAndHandlerMap)
}

func (gr *routerGroup) GET(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodGet, handlers...)
	return gr
}

func (gr *routerGroup) POST(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodPost, handlers...)
	return gr
}

func (gr *routerGroup) PUT(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodPut, handlers...)
	return gr
}

func (gr *routerGroup) PATCH(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodPatch, handlers...)
	return gr
}

func (gr *routerGroup) HEAD(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodHead, handlers...)
	return gr
}

func (gr *routerGroup) OPTIONS(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodOptions, handlers...)
	return gr
}

func (gr *routerGroup) DELETE(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	gr.insert(route, http.MethodDelete, handlers...)
	return gr
}

func (gr *routerGroup) UseRouter(args ...interface{}) Controller {
	parentRoute, sourceRouters := useRouter(args...)
	mergeRouterWithRouter(parentRoute, &gr.router, sourceRouters...)
	return gr
}

func (gr *routerGroup) UseMiddleware(args ...interface{}) Controller {
	var totalArg int = len(args)

	if totalArg == 0 {
		panic("UseMiddleware must pass arguments")
	}

	var parentRoute string

	for index, arg := range args {
		var isFirstArg bool = index == 0

		switch arg.(type) {
		case string:
			if isFirstArg {
				if totalArg <= 1 {
					panic("UseMiddleware need atleast a handler")
				}
				parentRoute = formatRoute(arg.(string))
			} else {
				panic("UseMiddleware only accepts string as first argument")
			}
			break

		case Handler:

			break
		}
	}

	fmt.Println(parentRoute)

	return gr
}
