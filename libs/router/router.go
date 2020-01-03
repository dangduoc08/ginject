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
	handleRequest func(req *Request, res Response)
	isEnd         bool
}

type Request struct {
	*http.Request
	Params map[string]string
}

type Response interface {
	http.ResponseWriter
}

type RequestHandler interface {
	Get(path string, handleRequest func(req *Request, res Response))
	Post(path string, handleRequest func(req *Request, res Response))
	Put(path string, handleRequest func(req *Request, res Response))
	Delete(path string, handleRequest func(req *Request, res Response))
}

type router struct {
	RouterTree *tree
}
