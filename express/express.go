package express

import (
	"fmt"
	"net/http"
	"sync"
)

const (
	get     = "GET"
	post    = "POST"
	put     = "PUT"
	delete  = "DELETE"
	options = "OPTIONS"
	patch   = "PATCH"
	head    = "HEAD"
)

type Express struct {
	routerTree *trie
}

type Router interface {
	Get(path string, handleRequest func(req *Request, res ResponseExtender)) Router
	Post(path string, handleRequest func(req *Request, res ResponseExtender)) Router
	Put(path string, handleRequest func(req *Request, res ResponseExtender)) Router
	Delete(path string, handleRequest func(req *Request, res ResponseExtender)) Router
}

var instance *Express
var once sync.Once

// Init Express by implement thread safe singleton
func Init() *Express {
	if instance == nil {
		once.Do(func() {
			instance = new(Express)
			var newTrie *trie = new(trie)
			newTrie.node = make(map[string]*trie)
			instance.routerTree = newTrie
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				params := make(map[string]string)
				matched, handleRequest := instance.routerTree.match(r.URL.Path, r.Method, &params)
				if matched {
					req := Request{
						Request: r,
						Params:  params,
					}
					res := response{w}
					var resExt ResponseExtender = &res

					handleRequest(&req, resExt)
				} else {
					w.WriteHeader(404)
					fmt.Fprintf(w, "<!DOCTYPE html>"+
						"<html lang='en'>"+
						"<head>"+
						"<meta charset='utf-8'>"+
						"<title>Error</title>"+
						"</head>"+
						"<body>"+
						"<pre>Cannot %s %s</pre>"+
						"</body>"+
						"</html>",
						r.Method, r.URL.Path)
				}
			})
		})
	}
	return instance
}

func (express *Express) Get(path string, handleRequest func(req *Request, res ResponseExtender)) Router {
	express.routerTree.insert(path, get, handleRequest)
	return express
}

func (express *Express) Post(path string, handleRequest func(req *Request, res ResponseExtender)) Router {
	express.routerTree.insert(path, post, handleRequest)
	return express
}

func (express *Express) Put(path string, handleRequest func(req *Request, res ResponseExtender)) Router {
	express.routerTree.insert(path, put, handleRequest)
	return express
}

func (express *Express) Delete(path string, handleRequest func(req *Request, res ResponseExtender)) Router {
	express.routerTree.insert(path, delete, handleRequest)
	return express
}
