package routing

import (
	"net/http"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/utils"
)

type Router interface {
	Get(string, ...context.Handler) *Route
	Head(string, ...context.Handler) *Route
	Post(string, ...context.Handler) *Route
	Put(string, ...context.Handler) *Route
	Patch(string, ...context.Handler) *Route
	Delete(string, ...context.Handler) *Route
	Connect(string, ...context.Handler) *Route
	Options(string, ...context.Handler) *Route
	Trace(string, ...context.Handler) *Route
	All(string, ...context.Handler) *Route
	Group(prefix string, subRouters ...*Route) *Route
	Use(...context.Handler) *Route
	For(string) func(...context.Handler) *Route
}

var HTTP_METHODS = []string{
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

func (r *Route) Get(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodGet), http.MethodGet, handlers...)
}

func (r *Route) Head(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodHead), http.MethodHead, handlers...)
}

func (r *Route) Post(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodPost), http.MethodPost, handlers...)
}

func (r *Route) Put(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodPut), http.MethodPut, handlers...)
}

func (r *Route) Patch(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodPatch), http.MethodPatch, handlers...)
}

func (r *Route) Delete(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodDelete), http.MethodDelete, handlers...)
}

func (r *Route) Connect(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodConnect), http.MethodConnect, handlers...)
}

func (r *Route) Options(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodOptions), http.MethodOptions, handlers...)
}

func (r *Route) Trace(path string, handlers ...context.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodTrace), http.MethodTrace, handlers...)
}

func (r *Route) All(path string, handlers ...context.Handler) *Route {
	for _, method := range HTTP_METHODS {
		r.Add(AddMethodToRoute(path, method), method, handlers...)
	}

	return r
}

func (r *Route) Group(prefix string, subRouters ...*Route) *Route {
	if prefix == "" {
		prefix = "/"
	}

	// prevent add prefix include slash at last
	prefix = utils.StrRemoveEnd(prefix, "/")

	for _, subRouter := range subRouters {
		subRouter.scan(func(seg string, node *Trie) {
			r.Add(prefix+subRouter.List[node.Index], fromPatternToMethod(seg), node.Handlers...)
		})

		// add all injectable handlers from sub routers
		// to main router
		for route, injectableHandler := range subRouter.InjectableHandlers {
			r.InjectableHandlers[prefix+route] = injectableHandler
		}
	}

	return r
}

func (r *Route) Use(handlers ...context.Handler) *Route {
	r.Middlewares = append(r.Middlewares, handlers...)
	r.scan(func(seg string, node *Trie) {
		r.Add(r.List[node.Index], fromPatternToMethod(seg), handlers...)
	})

	return r
}

func (r *Route) For(path string, inclusions []string) func(handlers ...context.Handler) *Route {
	return func(handlers ...context.Handler) *Route {
		for _, method := range inclusions {
			r.Add(AddMethodToRoute(path, method), method, handlers...)
		}

		return r
	}
}

func (r *Route) Match(path, method string) (bool, string, map[string][]int, []string, []context.Handler) {
	return r.match(AddMethodToRoute(path, method), method)
}

func (r *Route) Range(cb func(method, route string)) {
	for _, r := range r.List {
		cb(SplitRoute(r))
	}
}
