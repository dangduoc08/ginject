package gogo

import (
	"fmt"
	"strings"
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
// 	router := {
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

// Create a router
// init empty slice base on
// HTTP methods
func newRouter() router {
	var r router = make(map[string][]map[int]interface{})
	var routeAndHandlerMapSlice []map[int]interface{}
	for _, httpMethod := range httpMethods {
		r[httpMethod] = routeAndHandlerMapSlice
	}
	return r
}

// Find index of existing route
// base on HTTP methods and route
func (r router) match(httpMethod, route string) int {
	routeAndHandlerMapSlice := r[httpMethod]
	var position int = -1

	for index, routeAndHandlerMap := range routeAndHandlerMapSlice {
		if route == routeAndHandlerMap[routeKey] {
			position = index
			break
		}
	}

	return position
}

// Push routes, handlers to router group
// if route was inited before, we will append handler
// else append new route and handler map
func (r router) insert(route, httpMethod string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Nil handler")
	}
	routeAndHandlerMapSlice := r[httpMethod]
	position := r.match(route, httpMethod)

	if position > -1 {

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
		r[httpMethod] = append(r[httpMethod], newRouteAndHandlerMap)
	}
}

func (r router) generate(route, httpMethod string) string {
	routeAndHandlerMapSlice := r[httpMethod]

	for _, routeAndHandlerMap := range routeAndHandlerMapSlice {
		var existingRoute string = routeAndHandlerMap[routeKey].(string)
		fmt.Println("Pushed", existingRoute[1:])
		fmt.Println("Pushed after split", strings.Split(existingRoute[1:], "/"))
		fmt.Println("Will push", route)
		fmt.Println("Will push after split", strings.Split(route, "/"))
	}

	return ""
}
