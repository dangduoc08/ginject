package common

import (
	"reflect"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

func (c *Control) addToRouters(path, method string, handlers ...ctx.Handler) {
	if reflect.ValueOf(c.routerMap).IsNil() {
		c.routerMap = make(map[string][]ctx.Handler)
	}
	prefix := ""
	for _, str := range c.prefixes {
		prefix += utils.StrAddBegin(utils.StrRemoveEnd(str, "/"), "/")
	}
	c.routerMap[routing.AddMethodToRoute(prefix+routing.ToEndpoint(path), method)] = handlers
}
