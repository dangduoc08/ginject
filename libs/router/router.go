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

type response struct {
	http.ResponseWriter
}

type router struct {
	routerTree *tree
}

type ResponseExtender interface {
	http.ResponseWriter
	Send(content string, arguments ...interface{}) ResponseExtender
	Status(statusCode int) ResponseExtender
}

type RequestHandler interface {
	Get(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler
	Post(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler
	Put(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler
	Delete(path string, handleRequest func(req *Request, res ResponseExtender)) RequestHandler
}
