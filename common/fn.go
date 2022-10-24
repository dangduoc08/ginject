package common

import (
	"reflect"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

func (c *Control) addToRouters(path, method string, handlers ...ctx.Handler) {
	if reflect.ValueOf(c.routers).IsNil() {
		c.routers = make(map[string][]ctx.Handler)
	}
	prefix := ""
	for _, str := range c.prefixes {
		prefix += utils.StrAddBegin(utils.StrRemoveEnd(str, "/"), "/")
	}
	c.routers[routing.AddMethodToRoute(prefix+routing.ToEndpoint(path), method)] = handlers
}
