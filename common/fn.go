package common

import (
	"reflect"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

func (r *Rest) addToRouters(path, method string, handlers ...ctx.Handler) {
	if reflect.ValueOf(r.routerMap).IsNil() {
		r.routerMap = make(map[string][]ctx.Handler)
	}
	prefix := ""
	for _, str := range r.prefixes {
		prefix += utils.StrAddBegin(utils.StrRemoveEnd(str, "/"), "/")
	}
	r.routerMap[routing.AddMethodToRoute(prefix+routing.ToEndpoint(path), method)] = handlers
}
