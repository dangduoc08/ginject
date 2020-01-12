package router

func (r *router) Get(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler {
	r.routerTree.insert(path, get, handleRequest)
	return r
}

func (r *router) Post(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler {
	r.routerTree.insert(path, post, handleRequest)
	return r
}

func (r *router) Put(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler {
	r.routerTree.insert(path, put, handleRequest)
	return r
}

func (r *router) Delete(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler {
	r.routerTree.insert(path, delete, handleRequest)
	return r
}
