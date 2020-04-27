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
//					1st_Handler,
//					2nd_handler,
//					3rd_handler
//					...
//				]
//			}
//		]
//		...
//	}
type router map[string][]map[string]interface{}

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
	var r router = make(map[string][]map[string]interface{})
	routeAndHandlerArray := []map[string]interface{}{}

	for _, httpMethod := range httpMethods {
		r[httpMethod] = routeAndHandlerArray
	}

	return &r
}

// Push routes, handlers to router
func (r *router) insert(route, httpMethod string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Nil handler")
	}
	router := *r
	routeAndHandlerMap := make(map[string]interface{})
	routeAndHandlerMap[routeKey] = route
	routeAndHandlerMap[handlersKey] = handlers
	router[httpMethod] = append(router[httpMethod], routeAndHandlerMap)
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

func (r *router) GET(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	r.insert(route, http.MethodGet, handlers...)
	return r
}

func (r *router) POST(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	r.insert(route, http.MethodPost, handlers...)
	return r
}

func (r *router) PUT(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	r.insert(route, http.MethodPut, handlers...)
	return r
}

func (r *router) PATCH(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	r.insert(route, http.MethodPatch, handlers...)
	return r
}

func (r *router) HEAD(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	r.insert(route, http.MethodHead, handlers...)
	return r
}

func (r *router) OPTIONS(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	r.insert(route, http.MethodOptions, handlers...)
	return r
}

func (r *router) DELETE(route string, handlers ...Handler) Controller {
	route = formatRoute(route)
	r.insert(route, http.MethodDelete, handlers...)
	return r
}

func (r *router) UseRouter(args ...interface{}) Controller {
	parentRoute, sourceRouters := useRouter(args...)
	mergeRouterWithRouter(parentRoute, r, sourceRouters...)
	return r
}

// Use method implements Controller interface
func (r *router) UseMiddleware(args ...interface{}) Controller {
	if len(args) == 0 {
		panic("Missing arguments")
	}

	// for index, arg := range args {
	// 	var isFirstArg bool = index == 0
	// 	if isFirstArg {
	// 		switch arg.(type) {
	// 		case string:
	// 			var parentRoute = arg.(string)
	// 			var sourceRouter []*router = args[1:].(*router)
	// 			r.mergeRouterWithRouter(parentRoute, sourceRouter...)
	// 			break
	// 		}
	// 	}
	// }

	return r
}
