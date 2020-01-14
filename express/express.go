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

type Handler func(req *Request, res ResponseExtender, next func()) // Function handles middlewares and request

type Router interface {
	Get(path string, handlers ...Handler) Router
	Post(path string, handlers ...Handler) Router
	Put(path string, handlers ...Handler) Router
	Delete(path string, handlers ...Handler) Router
}

var instance *Express
var once sync.Once

// Init Express by implement thread safe singleton
func Init() *Express {
	if instance == nil {
		// This function only call once time
		once.Do(func() {
			instance = new(Express)
			// Create a nil trie to insert routers
			var newTrie *trie = new(trie)
			newTrie.node = make(map[string]*trie)
			// Add trie to router tree
			instance.routerTree = newTrie
			// All request will be accept to this handle function
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				params := make(map[string]string)
				matched, handlers := instance.routerTree.match(r.URL.Path, r.Method, &params)
				// Router existed in trie
				if matched {
					// Add http.Request and params to req
					req := Request{
						Request: r,
						Params:  params,
					}
					// Override http.Responsewritter
					res := response{w}
					var resExt ResponseExtender = &res
					// Handle middleware
					// treat the last handler is handle request function
					// any handlers not the last is middleware function
					var handleRequestIndex int = len(handlers) - 1
					var isNextCalled bool
					for index, handlerFn := range handlers {
						// Handle middleware
						if index != handleRequestIndex {
							isNextCalled = false
							// Because it is middleware function
							// therefore pass next function
							// if next function was invoked in router
							// handler will move to the next handler function
							handlerFn(&req, resExt, func() { isNextCalled = true })
							if !isNextCalled {
								break
							}
						} else {
							// Because it is the last handle request function
							// therefore did not pass the next function
							handlerFn(&req, resExt, nil)
						}
					}
				} else {
					// If no route matched, it's will send HTML 404 page
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
