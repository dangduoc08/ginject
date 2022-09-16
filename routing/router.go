package routing

import (
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/utils"
)

type Router interface {
	Get(string, ...ctx.Handler) Router
	Head(string, ...ctx.Handler) Router
	Post(string, ...ctx.Handler) Router
	Put(string, ...ctx.Handler) Router
	Patch(string, ...ctx.Handler) Router
	Delete(string, ...ctx.Handler) Router
	Connect(string, ...ctx.Handler) Router
	Options(string, ...ctx.Handler) Router
	Trace(string, ...ctx.Handler) Router
	Group(prefix string, subRouters ...*Route) Router
	Use(...ctx.Handler) Router
	For(string) func(...ctx.Handler) Router
	Range(func(string, string))
	Match(string, string) (bool, string, map[string][]int, []string, []ctx.Handler)
}

func (r *Route) Get(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodGet), handlers...)
}

func (r *Route) Head(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodHead), handlers...)
}

func (r *Route) Post(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodPost), handlers...)
}

func (r *Route) Put(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodPut), handlers...)
}

func (r *Route) Patch(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodPatch), handlers...)
}

func (r *Route) Delete(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodDelete), handlers...)
}

func (r *Route) Connect(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodConnect), handlers...)
}

func (r *Route) Options(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodOptions), handlers...)
}

func (r *Route) Trace(path string, handlers ...ctx.Handler) Router {
	return r.add(addMethodToRoute(path, http.MethodTrace), handlers...)
}

func (r *Route) Group(prePath string, subRouters ...*Route) Router {
	if prePath == "" {
		prePath = "/"
	}

	// prevent add prePath include slash at last
	prePath = utils.StrRemoveEnd(prePath, "/")

	for _, subRouter := range subRouters {
		subRouter.scan(func(node *Trie) {
			r.add(prePath+subRouter.List[node.Index], node.Handlers...)
		})
	}

	return r
}

func (r *Route) Use(handlers ...ctx.Handler) Router {
	r.Middlewares = append(r.Middlewares, handlers...)
	r.scan(func(node *Trie) {
		r.add(r.List[node.Index], handlers...)
	})

	return r
}

func (r *Route) For(path string) func(handlers ...ctx.Handler) Router {
	return func(handlers ...ctx.Handler) Router {
		for _, method := range HTTP_METHODS {
			r.add(addMethodToRoute(path, method), handlers...)
		}

		return r
	}
}

func (r *Route) Match(path, method string) (bool, string, map[string][]int, []string, []ctx.Handler) {
	return r.match(addMethodToRoute(path, method))
}

func (r *Route) Range(cb func(method, route string)) {
	for _, r := range r.List {
		cb(splitRoute(r))
	}
}
