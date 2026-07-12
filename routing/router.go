package routing

import (
	"errors"
	"net/http"
	"path"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/ds"
	"github.com/dangduoc08/ginject/internal/str"
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

// RouterItem holds everything Match needs once the trie has resolved a
// path: which method/version it was registered for, and the handler chain.
// Several RouterItems can share the same trie leaf (same Index, same route)
// when the same path is registered under different methods/versions.
type RouterItem struct {
	Method       string
	Version      string
	Pattern      string
	Index        int
	HandlerIndex int
	Handlers     []ctx.Handler
	ParamKeys    map[string][]int
}

func findRouterItem(items []RouterItem, method, version string) (RouterItem, bool) {
	for _, item := range items {
		if item.Method == method && item.Version == version {
			return item, true
		}
	}
	return RouterItem{}, false
}

type Router struct {
	trie               *ds.Trie
	Hash               map[string][]RouterItem
	routePatterns      []string
	GlobalMiddlewares  []ctx.Handler
	InjectableHandlers map[string]any
}

func NewRouter() *Router {
	return &Router{
		trie:               ds.NewTrie(),
		Hash:               make(map[string][]RouterItem),
		GlobalMiddlewares:  []ctx.Handler{},
		InjectableHandlers: make(map[string]any),
	}
}

func (r *Router) push(method, route, version string, caller int, handlers ...ctx.Handler) *Router {
	routePattern := str.Enclose(route, '/')

	items := r.Hash[routePattern]
	itemPos := -1
	for i := range items {
		if items[i].Method == method && items[i].Version == version {
			itemPos = i
			break
		}
	}

	var item RouterItem
	if itemPos == -1 {
		var routeIndex int
		if len(items) > 0 {
			routeIndex = items[0].Index
		} else {
			r.routePatterns = append(r.routePatterns, routePattern)
			routeIndex = len(r.routePatterns) - 1
		}
		item = RouterItem{
			Method:       method,
			Version:      version,
			Pattern:      MethodRouteVersionToPattern(method, route, version),
			Index:        routeIndex,
			HandlerIndex: -1,
		}
	} else {
		item = items[itemPos]
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

	parsedRoute, paramKey := ParseToParamKey(routePattern)
	if len(paramKey) > 0 {
		item.ParamKeys = paramKey
	}

	if itemPos == -1 {
		items = append(items, item)
	} else {
		items[itemPos] = item
	}
	r.Hash[routePattern] = items

	r.trie.Insert(routePattern, parsedRoute, '/')

	return r
}

func (r *Router) Match(method, route, version string) (bool, string, map[string][]int, []string, []ctx.Handler) {
	searchPath := str.Enclose(path.Clean(route), '/')
	matchedRaw, wildcardRaw, paramVals := r.trie.Find(searchPath, '/', true)

	item, ok := findRouterItem(r.Hash[matchedRaw], method, version)
	if !ok {
		item, ok = findRouterItem(r.Hash[wildcardRaw], method, version)
	}
	if !ok {
		return false, "", nil, nil, nil
	}

	return true, item.Pattern, item.ParamKeys, paramVals, item.Handlers
}

func (r *Router) Group(prefix string, subRouters ...*Router) *Router {
	for _, subRouter := range subRouters {
		for route, items := range subRouter.Hash {
			groupPath := prefix + route

			for _, routerItem := range items {
				if routerItem.HandlerIndex > -1 {
					groupPathPattern := str.Enclose(groupPath, '/')
					r.routePatterns = append(r.routePatterns, groupPathPattern)
					r.Hash[groupPathPattern] = append(r.Hash[groupPathPattern], RouterItem{
						Method:       routerItem.Method,
						Version:      routerItem.Version,
						Pattern:      MethodRouteVersionToPattern(routerItem.Method, groupPath, routerItem.Version),
						Index:        len(r.routePatterns) - 1,
						HandlerIndex: routerItem.HandlerIndex,
					})
				}

				handlers := append(r.GlobalMiddlewares[:len(r.GlobalMiddlewares):len(r.GlobalMiddlewares)], routerItem.Handlers...)
				r.push(routerItem.Method, groupPath, routerItem.Version, GROUP, handlers...)
			}
		}

		for route, injectableHandler := range subRouter.InjectableHandlers {
			r.InjectableHandlers[str.Enclose(prefix+route, '/')] = injectableHandler
		}
	}

	return r
}

func (r *Router) Use(handlers ...ctx.Handler) *Router {

	// use for global middlewares
	// once no route matched
	// this middlewares still need invoking
	r.GlobalMiddlewares = append(r.GlobalMiddlewares, handlers...)

	for route, items := range r.Hash {
		for _, routerItem := range items {
			r.push(routerItem.Method, route, routerItem.Version, USE, handlers...)
		}
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
	var handlerKind reflect.Kind
	if handler != nil {
		handlerKind = reflect.TypeOf(handler).Kind()
	}
	if handler == nil || handlerKind != reflect.Func {
		panic(errors.New(
			color.FmtRed(
				"invalid handler: %v is not a handler",
				handlerKind,
			),
		))
	}

	r.InjectableHandlers[MethodRouteVersionToPattern(method, route, version)] = handler
	r.Add(method, route, version, nil)

	return r
}
