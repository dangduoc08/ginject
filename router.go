package express

import "net/http"

type Router interface {
	Get(path string, handlers ...Handler) Router
	Post(path string, handlers ...Handler) Router
	Put(path string, handlers ...Handler) Router
	Delete(path string, handlers ...Handler) Router
}

func (express *Express) Get(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, http.MethodGet, handlers...)
	return express
}

func (express *Express) Post(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, http.MethodPost, handlers...)
	return express
}

func (express *Express) Put(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, http.MethodPut, handlers...)
	return express
}

func (express *Express) Delete(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, http.MethodDelete, handlers...)
	return express
}
