package router

import "net/http"

const (
	get     = "GET"
	post    = "POST"
	put     = "PUT"
	delete  = "DELETE"
	options = "OPTIONS"
	patch   = "PATCH"
	head    = "HEAD"
)

type tree struct {
	node          map[string]*tree
	params        map[string]string
	httpMethod    map[string]bool
	handleRequest func(req *Request, res ResponseExtender)
	isEnd         bool
}

type Request struct {
	*http.Request
	Params map[string]string
}

type ResponseExtender interface {
	http.ResponseWriter
	Send(content string, arguments ...interface{})
}

type response struct {
	http.ResponseWriter
}

type RequestHandler interface {
	Get(path string, handleRequest func(req *Request, res ResponseExtender))
	Post(path string, handleRequest func(req *Request, res ResponseExtender))
	Put(path string, handleRequest func(req *Request, res ResponseExtender))
	Delete(path string, handleRequest func(req *Request, res ResponseExtender))
}

type router struct {
	routerTree *tree
}
