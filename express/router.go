package express

type Router interface {
	Get(path string, handlers ...Handler) Router
	Post(path string, handlers ...Handler) Router
	Put(path string, handlers ...Handler) Router
	Delete(path string, handlers ...Handler) Router
}

func (express *Express) Get(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, get, handlers...)
	return express
}

func (express *Express) Post(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, post, handlers...)
	return express
}

func (express *Express) Put(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, put, handlers...)
	return express
}

func (express *Express) Delete(path string, handlers ...Handler) Router {
	express.routerTree.insert(path, delete, handlers...)
	return express
}
