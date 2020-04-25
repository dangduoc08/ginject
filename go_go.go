package gogo

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// G struct holds
// prefix-tree data structure
// with prefix-tree, the time complex
// when iterable trie to match router
// will be n = len(route)
type G struct {
	routerTree *trie
}

// Handler handle request and response
// with third param is a next function,
// we can use as a middleware function
// by pass many handler arguments
// and invoke next function
type Handler func(req *Request, res ResponseExtender, next func())

var instance *G
var once sync.Once

// GoGo init app by implement thread safe singleton
func GoGo() *G {
	if instance == nil {
		once.Do(func() {
			instance = new(G)

			// Create a nil trie to insert routers
			var newTrie *trie = new(trie)
			newTrie.node = make(map[string]*trie)
			instance.routerTree = newTrie

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				params := make(map[string]string)
				matched, handlers := instance.routerTree.match(r.URL.Path, r.Method, &params)

				// Router existed in trie
				if matched {

					// Add http.Request, params and context to *http.Request
					req := Request{
						Request: r,
						Params:  params,
						ctx:     context.Background(),
					}

					// Override http.Responsewritter interface
					res := response{w}
					var resExternder ResponseExtender = &res

					// Rule to handles middleware functions:
					// other handlers are middleware handlers
					// last handler is main response handler
					var handleRequestIndex int = len(handlers) - 1
					for index, handlerFn := range handlers {
						if index != handleRequestIndex {
							var isNextCalled bool

							// Because it is middleware function
							// it's will pass next function to third argument
							// if next function was invoked in router
							// handler will move to the next functions
							handlerFn(&req, resExternder, func() { isNextCalled = true })
							if !isNextCalled {
								break
							}
						} else {

							// Because it is the last handler
							// it's won't pass the next function
							handlerFn(&req, resExternder, nil)
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

// GET method
func (gg *G) GET(route string, handlers ...Handler) *G {
	gg.routerTree.insert(route, http.MethodGet, handlers...)
	return gg
}

// POST method
func (gg *G) POST(route string, handlers ...Handler) *G {
	gg.routerTree.insert(route, http.MethodPost, handlers...)
	return gg
}

// PUT method
func (gg *G) PUT(route string, handlers ...Handler) *G {
	gg.routerTree.insert(route, http.MethodPut, handlers...)
	return gg
}

// DELETE method
func (gg *G) DELETE(route string, handlers ...Handler) *G {
	gg.routerTree.insert(route, http.MethodDelete, handlers...)
	return gg
}
