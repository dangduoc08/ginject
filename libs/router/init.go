package router

import (
	"fmt"
	"net/http"
	"sync"
)

var instance *router

// Init router by implement thread safe singleton
func Init() *router {
	var once sync.Once
	if instance == nil {
		once.Do(func() {
			instance = new(router)
			var trie *tree = new(tree)
			trie.node = make(map[string]*tree)
			instance.routerTree = trie
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
