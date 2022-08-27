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
	rd := &routerData{
		Handlers: &handlers,
		Vars:     vars,
	}

	a.trie.insert(routeWithVar, len(a.array))
	a.array = append(a.array, map[string]*routerData{
		handledRoute: rd,
	})

	return a
}

func (a *adapter) find(route string) (bool, string, *routerData) {
	handledRoute := handleRoute(route)
	shadowRd := new(routerData)
	_, shadowRd.Vars = core.NewVar("")
	shadowRd.Handlers = &[]core.Handler{}

	if isFound, index, varParams := a.trie.find(handledRoute); isFound &&
		index > -1 {
		for route, routerData := range a.array[index] {

			// bind handler functions
			shadowRd.Handlers = routerData.Handlers
			for k, v := range routerData.Vars.KeyValue {
				i := v.(int)
				if varParams[i] != "" {

					// bind value to vars
					shadowRd.Vars.Set(k, varParams[i])
				}
			}
			return isFound, route, shadowRd
		}
	}

	return false, "", shadowRd
}
