package routing

import (
	"github.com/dangduoc08/go-go/core"
	"github.com/dangduoc08/go-go/helper"
)

type adapter struct {
	*Router
}

func handleRoute(route string) string {
	return helper.AddAtEnd(helper.AddAtBegin(helper.RemoveSpace(route), helper.SLASH), helper.SLASH)
}

func (a *adapter) insert(route string, handlers ...core.Handler) *adapter {
	handledRoute := handleRoute(route)
	routeWithVar, vars := core.NewVar(handledRoute)

	a.trie.insert(routeWithVar, &routerData{
		Handlers: &handlers,
		Vars:     vars,
	})

	return a
}

func (a *adapter) find(route string) (bool, *routerData) {
	handledRoute := handleRoute(route)
	shadowRd := new(routerData)
	_, shadowRd.Vars = core.NewVar("")
	shadowRd.Handlers = &[]core.Handler{}

	if isFound, varParams, rD := a.trie.find(handledRoute); isFound &&
		rD != nil &&
		rD.Vars != nil &&
		rD.Vars.KeyValue != nil {

		// bind handler functions
		shadowRd.Handlers = rD.Handlers
		for k, v := range rD.Vars.KeyValue {
			i := v.(int)
			if varParams[i] != "" {

				// bind value to vars
				shadowRd.Vars.Set(k, varParams[i])
			}
		}

		return isFound, shadowRd
	}

	return false, shadowRd
}
