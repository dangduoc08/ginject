package gogo

import "net/http"

// Router defines all supported http method handlers
type Router interface {
	Get(path string, handlers ...Handler) Router
	Post(path string, handlers ...Handler) Router
	Put(path string, handlers ...Handler) Router
	Delete(path string, handlers ...Handler) Router
}

// Get method
func (gg *GoGo) Get(path string, handlers ...Handler) Router {
	gg.routerTree.insert(path, http.MethodGet, handlers...)
	return gg
}

// Post method
func (gg *GoGo) Post(path string, handlers ...Handler) Router {
	gg.routerTree.insert(path, http.MethodPost, handlers...)
	return gg
}

// Put method
func (gg *GoGo) Put(path string, handlers ...Handler) Router {
	gg.routerTree.insert(path, http.MethodPut, handlers...)
	return gg
}

// Delete method
func (gg *GoGo) Delete(path string, handlers ...Handler) Router {
	gg.routerTree.insert(path, http.MethodDelete, handlers...)
	return gg
}
