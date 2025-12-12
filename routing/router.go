package routing

import (
	"errors"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/utils"
)

const SERVE = "SERVE" // Serving static files directive

var OperationsMapHTTPMethods = map[string]string{
	http.MethodGet:     http.MethodGet,
	http.MethodHead:    http.MethodHead,
	http.MethodPost:    http.MethodPost,
	http.MethodPut:     http.MethodPut,
	http.MethodPatch:   http.MethodPatch,
	http.MethodDelete:  http.MethodDelete,
	http.MethodConnect: http.MethodConnect,
	http.MethodOptions: http.MethodOptions,
	http.MethodTrace:   http.MethodTrace,
	SERVE:              http.MethodGet,
}

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
	SERVE,
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

func (r *Router) push(method, route, version string, caller int, handlers ...ctx.Handler) *Router {
	endpoint := MethodRouteVersionToPattern(method, route, version)

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

func (r *Router) Match(method, route, version string) (bool, string, map[string][]int, []string, []ctx.Handler) {
	route = strings.Join([]string{filepath.Clean(route), "/|", version, "|/[", method, "]/"}, "")
	if matchedRouterHash, ok := r.Hash[route]; ok && !matchedRouterHash.isRouteContainsParams {
		return ok, route, nil, nil, matchedRouterHash.Handlers
	}

	i, paramKeys, paramVals, handlers := r.Trie.find(route, method, version, '/')
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
			method, path, version := PatternToMethodRouteVersion(route)
			groupPath := prefix + path

			if routerItem.HandlerIndex > -1 {
				endpoint := MethodRouteVersionToPattern(method, groupPath, version)
				r.List = append(r.List, endpoint)
				r.Hash[endpoint] = RouterItem{
					Index:        len(r.List) - 1,
					HandlerIndex: routerItem.HandlerIndex,
				}
			}

			handlers := append(r.GlobalMiddlewares, routerItem.Handlers...)
			r.push(method, groupPath, version, GROUP, handlers...)
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
		method, path, version := PatternToMethodRouteVersion(route)
		r.push(method, path, version, USE, handlers...)
	}

	return r
}

func (r *Router) For(methodInclusions []string, route string, version string) func(handlers ...ctx.Handler) *Router {
	return func(handlers ...ctx.Handler) *Router {
		for _, method := range methodInclusions {
			r.push(method, route, version, FOR, handlers...)
		}

		return r
	}
}

// alway use latest add
func (r *Router) Add(method, route, version string, handler ctx.Handler) *Router {
	r.push(method, route, version, ADD, handler)

	return r
}

func (r *Router) AddInjectableHandler(method, route, version string, handler any) *Router {
	handlerKind := reflect.TypeOf(handler).Kind()
	if handler == nil || handlerKind != reflect.Func {
		panic(errors.New(
			utils.FmtRed(
				"%v is not a handler",
				handlerKind,
			),
		))
	}

	r.InjectableHandlers[MethodRouteVersionToPattern(method, route, version)] = handler
	r.Add(method, route, version, nil)

	return r
}
