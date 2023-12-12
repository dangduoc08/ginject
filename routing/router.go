package routing

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/utils"
)

var HTTPMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodConnect,
	http.MethodOptions,
	http.MethodTrace,
}

const (
	ADD = iota + 1
	USE
	FOR
	GROUP
)

type RouterItem struct {
	Index                 int
	HandlerIndex          int
	Handlers              []ctx.Handler
	isRouteContainsParams bool
}

type Router struct {
	*Trie
	Hash               map[string]RouterItem
	List               []string
	GlobalMiddlewares  []ctx.Handler
	InjectableHandlers map[string]any
}

func NewRouter() *Router {
	return &Router{
		Trie:               NewTrie(),
		Hash:               make(map[string]RouterItem),
		GlobalMiddlewares:  []ctx.Handler{},
		InjectableHandlers: make(map[string]any),
	}
}

func (r *Router) push(route, method string, caller int, handlers ...ctx.Handler) *Router {
	endpoint := ToEndpoint(AddMethodToRoute(route, method))
	var item RouterItem

	if matchedRouterHash, ok := r.Hash[endpoint]; !ok {
		r.List = append(r.List, endpoint)
		item.Index = len(r.List) - 1
		item.HandlerIndex = -1
	} else {
		item = r.Hash[endpoint]
		item.Index = matchedRouterHash.Index
	}

	handlerTotal := len(item.Handlers)
	globalMiddlewareTotal := len(r.GlobalMiddlewares)

	if caller == USE || caller == GROUP {

		// USE never has handlerTotal == 0 case
		// check line 179
		item.Handlers = append(item.Handlers, handlers...)
	}

	if caller == FOR {

		// handle case
		// USE called first
		// FOR called later
		if handlerTotal == 0 && globalMiddlewareTotal > 0 {
			item.Handlers = append(item.Handlers, r.GlobalMiddlewares...)
			item.Handlers = append(item.Handlers, handlers...)
		} else {
			item.Handlers = append(item.Handlers, handlers...)
		}
	}

	if caller == ADD {

		// ADD call first
		// USE call later
		if handlerTotal == 0 && globalMiddlewareTotal == 0 {

			item.Handlers = append(item.Handlers, handlers...)
			item.HandlerIndex = 0

			// USE call first
			// ADD call later
		} else if handlerTotal == 0 && globalMiddlewareTotal > 0 {

			// handler hasn't added yet
			item.Handlers = append(item.Handlers, r.GlobalMiddlewares...)
			item.Handlers = append(item.Handlers, handlers...)
			item.HandlerIndex = globalMiddlewareTotal
		} else if item.HandlerIndex > -1 {
			// handler was added before

			// remove the current
			// append new one
			item.Handlers = append(item.Handlers[:item.HandlerIndex], item.Handlers[item.HandlerIndex+1:]...)
			item.Handlers = append(item.Handlers, handlers...)
			item.HandlerIndex = handlerTotal - 1
		} else if item.HandlerIndex < 0 {

			// handler hasn't added yet
			item.HandlerIndex = handlerTotal
			item.Handlers = append(item.Handlers, handlers...)
		}
	}

	parsedRoute, paramKey := ParseToParamKey(endpoint)
	item.isRouteContainsParams = checkRouteContainsParams(parsedRoute)
	r.Hash[endpoint] = item
	r.Trie.insert(parsedRoute, '/', r.Hash[endpoint].Index, paramKey, r.Hash[endpoint].Handlers)

	return r
}

func (r *Router) Match(route, method string) (bool, string, map[string][]int, []string, []ctx.Handler) {
	route = AddMethodToRoute(route, method)

	if matchedRouterHash, ok := r.Hash[route]; ok && !matchedRouterHash.isRouteContainsParams {
		return ok, route, nil, nil, matchedRouterHash.Handlers
	}

	i, paramKeys, paramVals, handlers := r.Trie.find(ToEndpoint(route), method, '/')
	matchedRoute := ""
	isMatched := false
	if i > -1 {
		isMatched = true
		matchedRoute = r.List[i]
	}

	return isMatched, matchedRoute, paramKeys, paramVals, handlers
}

func (r *Router) Group(prefix string, subRouters ...*Router) *Router {
	for _, subRouter := range subRouters {
		for route, routerItem := range subRouter.Hash {
			method, path := SplitRoute(route)
			groupPath := prefix + path

			if routerItem.HandlerIndex > -1 {
				endpoint := ToEndpoint(AddMethodToRoute(groupPath, method))
				r.List = append(r.List, endpoint)
				r.Hash[endpoint] = RouterItem{
					Index:        len(r.List) - 1,
					HandlerIndex: routerItem.HandlerIndex,
				}
			}

			handlers := append(r.GlobalMiddlewares, routerItem.Handlers...)
			r.push(groupPath, method, GROUP, handlers...)
		}

		for route, injectableHandler := range subRouter.InjectableHandlers {
			r.InjectableHandlers[ToEndpoint(prefix+route)] = injectableHandler
		}
	}

	return r
}

func (r *Router) Use(handlers ...ctx.Handler) *Router {

	// use for global middlewares
	// once no route matched
	// this middlewares still need invoking
	r.GlobalMiddlewares = append(r.GlobalMiddlewares, handlers...)

	for route := range r.Hash {
		method, path := SplitRoute(route)
		r.push(path, method, USE, handlers...)
	}

	return r
}

func (r *Router) For(path string, inclusions []string) func(handlers ...ctx.Handler) *Router {
	return func(handlers ...ctx.Handler) *Router {
		for _, method := range inclusions {
			r.push(path, method, FOR, handlers...)
		}

		return r
	}
}

// alway use latest add
func (r *Router) Add(route, method string, handler ctx.Handler) *Router {
	r.push(route, method, ADD, handler)

	return r
}

func (r *Router) AddInjectableHandler(route, method string, handler any) *Router {
	handlerKind := reflect.TypeOf(handler).Kind()
	if handler == nil || handlerKind != reflect.Func {
		panic(fmt.Errorf(
			utils.FmtRed(
				"%v is not a handler",
				handlerKind,
			),
		))
	}

	r.InjectableHandlers[ToEndpoint(AddMethodToRoute(route, method))] = handler
	r.Add(route, method, nil)

	return r
}

func (r *Router) Range(cb func(method, route string)) {

	// only scan for main handlers
	for r, item := range r.Hash {
		if item.HandlerIndex > -1 {
			cb(SplitRoute(r))
		}
	}
}
