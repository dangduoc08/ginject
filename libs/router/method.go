package router

func (r *router) Get(path string, handleRequest func(req *Request, res Response)) {
	r.RouterTree.insert(path, get, handleRequest)
}

func (r *router) Post(path string, handleRequest func(req *Request, res Response)) {
	r.RouterTree.insert(path, post, handleRequest)
}

func (r *router) Put(path string, handleRequest func(req *Request, res Response)) {
	r.RouterTree.insert(path, put, handleRequest)
}

func (r *router) Delete(path string, handleRequest func(req *Request, res Response)) {
	r.RouterTree.insert(path, delete, handleRequest)
}
