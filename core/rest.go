package core

import (
	"net/http"

	"github.com/dangduoc08/gooh/ctx"
)

type Rest struct {
	prefixes  []string
	routerMap map[string][]ctx.Handler
}

func (r *Rest) Prefix(prefix string) *Rest {
	r.prefixes = append(r.prefixes, prefix)
	return r
}

func (r *Rest) Get(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodGet, handlers...)
	return r
}

func (r *Rest) Head(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodHead, handlers...)
	return r
}

func (r *Rest) Post(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodPost, handlers...)
	return r
}

func (r *Rest) Put(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodPut, handlers...)
	return r
}

func (r *Rest) Patch(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodPatch, handlers...)
	return r
}

func (r *Rest) Delete(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodDelete, handlers...)
	return r
}

func (r *Rest) Connect(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodDelete, handlers...)
	return r
}

func (r *Rest) Options(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodOptions, handlers...)
	return r
}

func (r *Rest) Trace(path string, handlers ...ctx.Handler) *Rest {
	r.addToRouters(path, http.MethodTrace, handlers...)
	return r
}
