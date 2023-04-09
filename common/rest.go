package common

import (
	"net/http"
	"reflect"

	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type Rest struct {
	prefixes  []string
	RouterMap map[string]any
}

func (r *Rest) addToRouters(path, method string, injectableHandler any) {
	if reflect.ValueOf(r.RouterMap).IsNil() {
		r.RouterMap = make(map[string]any)
	}
	prefix := ""
	for _, str := range r.prefixes {
		prefix += utils.StrAddBegin(utils.StrRemoveEnd(str, "/"), "/")
	}
	r.RouterMap[routing.AddMethodToRoute(prefix+routing.ToEndpoint(path), method)] = injectableHandler
}

func (r *Rest) Prefix(prefix string) *Rest {
	r.prefixes = append(r.prefixes, prefix)
	return r
}

func (r *Rest) Get(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodGet, injectableHandler)
	return r
}

func (r *Rest) Head(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodHead, injectableHandler)
	return r
}

func (r *Rest) Post(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodPost, injectableHandler)
	return r
}

func (r *Rest) Put(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodPut, injectableHandler)
	return r
}

func (r *Rest) Patch(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodPatch, injectableHandler)
	return r
}

func (r *Rest) Delete(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodDelete, injectableHandler)
	return r
}

func (r *Rest) Connect(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodDelete, injectableHandler)
	return r
}

func (r *Rest) Options(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodOptions, injectableHandler)
	return r
}

func (r *Rest) Trace(path string, injectableHandler any) *Rest {
	r.addToRouters(path, http.MethodTrace, injectableHandler)
	return r
}

func (r *Rest) All(path string, injectableHandler any) *Rest {
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
		r.addToRouters(path, method, injectableHandler)
	}

	return r
}
