package gogo

import "net/http"

// Router defines all supported http method handlers
type Router interface {
	GET(route string, handlers ...Handler) Router
	POST(route string, handlers ...Handler) Router
	PUT(route string, handlers ...Handler) Router
	DELETE(route string, handlers ...Handler) Router
}

// GET method
func (gg *GoGo) GET(route string, handlers ...Handler) Router {
	gg.routerTree.insert(route, http.MethodGet, handlers...)
	return gg
}

// POST method
func (gg *GoGo) POST(route string, handlers ...Handler) Router {
	gg.routerTree.insert(route, http.MethodPost, handlers...)
	return gg
}

// PUT method
func (gg *GoGo) PUT(route string, handlers ...Handler) Router {
	gg.routerTree.insert(route, http.MethodPut, handlers...)
	return gg
}

// DELETE method
func (gg *GoGo) DELETE(route string, handlers ...Handler) Router {
	gg.routerTree.insert(route, http.MethodDelete, handlers...)
	return gg
}
