package routing

import (
	"github.com/dangduoc08/gooh/core"
	dataStructure "github.com/dangduoc08/gooh/data-structure"
)

type adapter struct {
	*Router
}

func handleRoute(route string) string {
	return dataStructure.AddAtEnd(dataStructure.AddAtBegin(dataStructure.RemoveSpace(route), dataStructure.SLASH), dataStructure.SLASH)
}

func (a *adapter) insertMiddleware(route string, handlers ...core.Handler) *adapter {
	handledRoute := handleRoute(route)
	routeWithVar, vars := core.NewVar(handledRoute)
	rd := &routerData{
		Handlers: &handlers,
		Vars:     vars,
	}

	a.Trie.Insert(routeWithVar, len(a.array))
	a.array = append(a.array, map[string]*routerData{
		handledRoute: rd,
	})

	return a
}

func (a *adapter) insert(route string, handlers ...core.Handler) *adapter {
	handledRoute := handleRoute(route)
	routeWithVar, vars := core.NewVar(handledRoute)
	rd := &routerData{
		Handlers: &handlers,
		Vars:     vars,
	}

	a.Trie.Insert(routeWithVar, len(a.array))
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

	if isFound, index, varParams := a.Trie.Find(handledRoute); isFound &&
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
