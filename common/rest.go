package common

import (
	"net/http"
	"reflect"

	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type Rest struct {
	prefix    string
	RouterMap map[string]any
}

func (r *Rest) addToRouters(prefix, path, method string, injectableHandler any) {
	if reflect.ValueOf(r.RouterMap).IsNil() {
		r.RouterMap = make(map[string]any)
	}
	prefix = utils.StrAddBegin(utils.StrRemoveEnd(prefix, "/"), "/")
	r.RouterMap[routing.AddMethodToRoute(prefix+routing.ToEndpoint(path), method)] = injectableHandler
}

func (r *Rest) Prefix(prefix string) *Rest {
	r.prefix = prefix
	return r
}

func (r *Rest) Get(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodGet, injectableHandler)
	return r
}

func (r *Rest) Head(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodHead, injectableHandler)
	return r
}

func (r *Rest) Post(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodPost, injectableHandler)
	return r
}

func (r *Rest) Put(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodPut, injectableHandler)
	return r
}

func (r *Rest) Patch(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodPatch, injectableHandler)
	return r
}

func (r *Rest) Delete(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodDelete, injectableHandler)
	return r
}

func (r *Rest) Connect(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodDelete, injectableHandler)
	return r
}

func (r *Rest) Options(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodOptions, injectableHandler)
	return r
}

func (r *Rest) Trace(path string, injectableHandler any) *Rest {
	r.addToRouters(r.prefix, path, http.MethodTrace, injectableHandler)
	return r
}

func (r *Rest) All(path string, injectableHandler any) *Rest {
	for _, method := range routing.HTTP_METHODS {
		r.addToRouters(r.prefix, path, method, injectableHandler)
	}

	return r
}
