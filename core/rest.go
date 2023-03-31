package core

import (
	"net/http"
	"reflect"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type Rest struct {
	prefixes  []string
	routerMap map[string][]context.Handler
}

func (r *Rest) addToRouters(path, method string, handlers ...context.Handler) {
	if reflect.ValueOf(r.routerMap).IsNil() {
		r.routerMap = make(map[string][]context.Handler)
	}
	prefix := ""
	for _, str := range r.prefixes {
		prefix += utils.StrAddBegin(utils.StrRemoveEnd(str, "/"), "/")
	}
	r.routerMap[routing.AddMethodToRoute(prefix+routing.ToEndpoint(path), method)] = handlers
}

func (r *Rest) Prefix(prefix string) *Rest {
	r.prefixes = append(r.prefixes, prefix)
	return r
}

func (r *Rest) Get(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodGet, handlers...)
	return r
}

func (r *Rest) Head(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodHead, handlers...)
	return r
}

func (r *Rest) Post(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodPost, handlers...)
	return r
}

func (r *Rest) Put(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodPut, handlers...)
	return r
}

func (r *Rest) Patch(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodPatch, handlers...)
	return r
}

func (r *Rest) Delete(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodDelete, handlers...)
	return r
}

func (r *Rest) Connect(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodDelete, handlers...)
	return r
}

func (r *Rest) Options(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodOptions, handlers...)
	return r
}

func (r *Rest) Trace(path string, handlers ...context.Handler) *Rest {
	r.addToRouters(path, http.MethodTrace, handlers...)
	return r
}

func (r *Rest) All(path string, handlers ...context.Handler) *Rest {
	httpMethods := [9]string{
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

	for _, method := range httpMethods {
		r.addToRouters(path, method, handlers...)
	}

	return r
}
