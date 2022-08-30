package routing

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/core"
	dataStructure "github.com/dangduoc08/gooh/data-structure"
)

type adapter struct {
	*Router
}

func handleRoute(route string) string {
	return dataStructure.AddAtEnd(dataStructure.AddAtBegin(dataStructure.RemoveSpace(route), dataStructure.SLASH), dataStructure.SLASH)
}

// func (adapterInstance *adapter) insertMiddleware(route string, handlers ...core.Handler) *adapter {
// 	handledRoute := handleRoute(route)
// 	routeWithParams, params := context.NewParam(handledRoute)
// 	routerDataInstance := &routerData{
// 		Handlers: &handlers,
// 		Params:   params,
// 	}

// 	adapterInstance.Trie.Insert(routeWithParams, len(adapterInstance.array))
// 	adapterInstance.array = append(adapterInstance.array, map[string]*routerData{
// 		handledRoute: routerDataInstance,
// 	})

// 	return adapterInstance
// }

func (adapterInstance *adapter) insert(route string, handlers ...core.Handler) *adapter {
	handledRoute := handleRoute(route)
	routeWithParams, params := context.NewParam(handledRoute)
	routerDataInstance := &routerData{
		Handlers: &handlers,
		Params:   params,
	}

	adapterInstance.Trie.Insert(routeWithParams, len(adapterInstance.array))
	adapterInstance.array = append(adapterInstance.array, map[string]*routerData{
		handledRoute: routerDataInstance,
	})

	return adapterInstance
}

func (adapterInstance *adapter) find(route string) (bool, string, *routerData) {
	handledRoute := handleRoute(route)
	shadowOfRouterData := new(routerData)
	_, shadowOfRouterData.Params = context.NewParam("")
	shadowOfRouterData.Handlers = &[]core.Handler{}

	if isEnd, index, paramValues := adapterInstance.Trie.Find(handledRoute); isEnd &&
		index > -1 {
		for route, routerData := range adapterInstance.array[index] {

			// bind handler functions
			shadowOfRouterData.Handlers = routerData.Handlers

			routerData.Params.ForEach(func(value interface{}, key string) {
				paramValue := paramValues[value.(int)]
				if paramValue != "" {

					// bind value to params
					shadowOfRouterData.Params.Set(key, paramValue)
				}
			})

			return isEnd, route, shadowOfRouterData
		}
	}

	return false, "", shadowOfRouterData
}
