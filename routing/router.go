package routing

import (
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/utils"
)

type Router interface {
	Get(string, ...ctx.Handler) *Route
	Head(string, ...ctx.Handler) *Route
	Post(string, ...ctx.Handler) *Route
	Put(string, ...ctx.Handler) *Route
	Patch(string, ...ctx.Handler) *Route
	Delete(string, ...ctx.Handler) *Route
	Connect(string, ...ctx.Handler) *Route
	Options(string, ...ctx.Handler) *Route
	Trace(string, ...ctx.Handler) *Route
	Group(prefix string, subRouters ...*Route) *Route
	Use(...ctx.Handler) *Route
	For(string) func(...ctx.Handler) *Route
}

func (r *Route) Get(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodGet), handlers...)
}

func (r *Route) Head(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodHead), handlers...)
}

func (r *Route) Post(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodPost), handlers...)
}

func (r *Route) Put(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodPut), handlers...)
}

func (r *Route) Patch(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodPatch), handlers...)
}

func (r *Route) Delete(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodDelete), handlers...)
}

func (r *Route) Connect(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodConnect), handlers...)
}

func (r *Route) Options(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodOptions), handlers...)
}

func (r *Route) Trace(path string, handlers ...ctx.Handler) *Route {
	return r.Add(AddMethodToRoute(path, http.MethodTrace), handlers...)
}

func (r *Route) Group(prefix string, subRouters ...*Route) *Route {
	if prefix == "" {
		prefix = "/"
	}

	// prevent add prefix include slash at last
	prefix = utils.StrRemoveEnd(prefix, "/")

	for _, subRouter := range subRouters {
		subRouter.scan(func(node *Trie) {
			r.Add(prefix+subRouter.List[node.Index], node.Handlers...)
		})
	}

	return r
}

func (r *Route) Use(handlers ...ctx.Handler) *Route {
	r.Middlewares = append(r.Middlewares, handlers...)
	r.scan(func(node *Trie) {
		r.Add(r.List[node.Index], handlers...)
	})

	return r
}

func (r *Route) For(path string) func(handlers ...ctx.Handler) *Route {
	return func(handlers ...ctx.Handler) *Route {
		for _, method := range HTTP_METHODS {
			r.Add(AddMethodToRoute(path, method), handlers...)
		}

		return r
	}
}

func (r *Route) Match(path, method string) (bool, string, map[string][]int, []string, []ctx.Handler) {
	return r.match(AddMethodToRoute(path, method))
}

func (r *Route) Range(cb func(method, route string)) {
	for _, r := range r.List {
		cb(splitRoute(r))
	}
}
