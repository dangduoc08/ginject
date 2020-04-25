package gogo

import (
	"net/http"
)

var httpMethods []string = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodDelete,
}

const (
	routeKey    = "route"
	handlersKey = "handlers"
)

// R holds struct:
//	{
//		'GET': [
//			{
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
type R map[string][]map[string]interface{}

type router interface {
	GET(route string, handlers ...Handler) *R
	POST(route string, handlers ...Handler) *R
	PUT(route string, handlers ...Handler) *R
	DELETE(route string, handlers ...Handler) *R
}

// Router init router struct
func Router() *R {
	var routers R = make(map[string][]map[string]interface{})
	var routeAndHandlerArray []map[string]interface{} = []map[string]interface{}{}

	for _, method := range httpMethods {
		routers[method] = routeAndHandlerArray
	}

	return &routers
}

// Insert helps push routes, handlers to router
func (r *R) insertRouter(method, route string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Nil handler")
	}
	route = handleSlash(route)
	routers := *r
	var routeAndHandlerMap map[string]interface{} = make(map[string]interface{})
	routeAndHandlerMap[routeKey] = route
	routeAndHandlerMap[handlersKey] = handlers
	routers[method] = append(routers[method], routeAndHandlerMap)
}

// GET method
func (r *R) GET(route string, handlers ...Handler) *R {
	r.insertRouter(http.MethodGet, route, handlers...)
	return r
}

// POST method
func (r *R) POST(route string, handlers ...Handler) *R {
	r.insertRouter(http.MethodPost, route, handlers...)
	return r
}

// PUT method
func (r *R) PUT(route string, handlers ...Handler) *R {
	r.insertRouter(http.MethodPut, route, handlers...)
	return r
}

// DELETE method
func (r *R) DELETE(route string, handlers ...Handler) *R {
	r.insertRouter(http.MethodDelete, route, handlers...)
	return r
}
