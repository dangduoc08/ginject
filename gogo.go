package gogo

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

// App struct holds
// prefix-tree data structure
// with prefix-tree, the time complex
// when iterable trie to match router
// will be n = len(route)
type app struct {
	routerTree *trie

	// Global middlewares
	middlewares []Handler
}

var instance *app
var once sync.Once

// GoGo inits app by implement thread safe singleton
// https://refactoring.guru/design-patterns/singleton
func GoGo() Controller {
	if instance == nil {
		once.Do(func() {
			instance = new(app)

			// Create a nil trie to insert routers
			var newTrie *trie = new(trie)
			newTrie.node = make(map[string]*trie)
			instance.routerTree = newTrie

			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

				// Override http.Responsewritter interface
				res := response{w}
				var resExternder ResponseExtender = &res

				// Add http.Request, params and context to *http.Request
				req := Request{
					Request: r,
					ctx:     context.Background(),
				}

				var isNextCalled bool

				// Invoke global middlewares
				// before run main handlers
				for _, middleware := range instance.middlewares {
					isNextCalled = false
					middleware(&req, resExternder, func() { isNextCalled = true })
					if !isNextCalled {
						break
					}
				}
				params := make(map[string]string)

				// Format route before find in trie
				var path string = r.Method + formatRoute(r.URL.Path)
				matched, handlers := instance.routerTree.match(path, r.Method, &params)

				// Router existed in trie
				if matched {
					req.Params = params

					// Rule to handles middleware functions:
					// other handlers are middleware handlers
					// last handler is main response handler
					var handleRequestIndex int = len(handlers) - 1
					for index, handlerFn := range handlers {
						if index != handleRequestIndex {
							isNextCalled = false

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
							// next function is not neccesery
							// but we still pass it to avoid point to nil error
							// when try to invoke its
							handlerFn(&req, resExternder, func() { isNextCalled = true })
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

func (gg *app) Get(route string, handlers ...Handler) Controller {
	route = http.MethodGet + formatRoute(route)
	gg.routerTree.insert(route, http.MethodGet, handlers...)
	return gg
}

func (gg *app) Post(route string, handlers ...Handler) Controller {
	route = http.MethodPost + formatRoute(route)
	gg.routerTree.insert(route, http.MethodPost, handlers...)
	return gg
}

func (gg *app) Put(route string, handlers ...Handler) Controller {
	route = http.MethodPut + formatRoute(route)
	gg.routerTree.insert(route, http.MethodPut, handlers...)
	return gg
}

func (gg *app) Delete(route string, handlers ...Handler) Controller {
	route = http.MethodDelete + formatRoute(route)
	gg.routerTree.insert(route, http.MethodDelete, handlers...)
	return gg
}

func (gg *app) Patch(route string, handlers ...Handler) Controller {
	route = http.MethodPatch + formatRoute(route)
	gg.routerTree.insert(route, http.MethodPatch, handlers...)
	return gg
}

func (gg *app) Head(route string, handlers ...Handler) Controller {
	route = http.MethodHead + formatRoute(route)
	gg.routerTree.insert(route, http.MethodHead, handlers...)
	return gg
}

func (gg *app) Options(route string, handlers ...Handler) Controller {
	route = http.MethodOptions + formatRoute(route)
	gg.routerTree.insert(route, http.MethodOptions, handlers...)
	return gg
}

func (gg *app) UseRouter(args ...interface{}) Controller {
	parentRoute, sourceRouterGroups := resolveRouterGroup(args...)
	mergeRouterWithApp(parentRoute, gg, sourceRouterGroups)
	return gg
}

func (gg *app) UseMiddleware(args ...interface{}) Controller {
	var totalArg int = len(args)

	if totalArg == 0 {
		panic("UseMiddleware must pass arguments")
	}

	var parentRoute string

	for index, arg := range args {
		var isFirstArg bool = index == 0

		switch arg.(type) {
		case string:
			if isFirstArg {
				if totalArg <= 1 {
					panic("UseMiddleware need atleast a handler")
				}
				parentRoute = formatRoute(arg.(string))
			} else {
				panic("UseMiddleware only accepts string as first argument")
			}
			break

		case Handler:

			break
		}
	}

	fmt.Println("app parentRoute", parentRoute)

	return gg
}
