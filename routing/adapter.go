package routing

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/ds"
)

const (
	ADD = iota + 1
	USE
)

type adapter struct {
	*Router
}

func (adapterInstance *adapter) serve(route string, whichMethodInvoke int, handlers ...context.Handler) *adapter {
	pushedHandledRoute := handlePath(route)

	if whichMethodInvoke == USE {
		isServeForAllRoutes := route == ds.WILDCARD

		// iterate all handlers array
		// if route matched or wildcard
		// append middlewares to handlers
		for _, routerDataMap := range adapterInstance.array {
			for cachedRoute, routerDataPt := range routerDataMap {
				if cachedRoute == pushedHandledRoute || isServeForAllRoutes {

					// append into all handler array
					*routerDataPt.Handlers = append(*routerDataPt.Handlers, handlers...)
				}
			}
		}
	} else if whichMethodInvoke == ADD {
		index := len(adapterInstance.middlewares) - 1

		// reversed iterate all middlewares array
		// if route matched or wildcard
		// prepend middlewares to handlers
		for index >= 0 {
			for middlewareCachedRoute, cachedMiddlewareArr := range adapterInstance.middlewares[index] {
				isServeForAllRoutes := middlewareCachedRoute == ds.WILDCARD
				for _, routerDataMap := range adapterInstance.array {
					for cachedRoute, routerDataPt := range routerDataMap {
						if cachedRoute == pushedHandledRoute && (cachedRoute == middlewareCachedRoute || isServeForAllRoutes) {

							// prepend into all handler array
							*routerDataPt.Handlers = append(*cachedMiddlewareArr, *routerDataPt.Handlers...)
						}
					}
				}
			}
			index--
		}
	}

	return adapterInstance
}

func (adapterInstance *adapter) insert(route string, handlers ...context.Handler) *adapter {
	handledRoute := handlePath(route)
	routeWithParams, params := context.NewParam(handledRoute)
	routerDataInstance := &routerData{
		Handlers: &handlers,
		Params:   params,
	}

	existingRouteIndex := ds.FindIndex(adapterInstance.array, func(elem map[string]*routerData, index int) bool {
		return elem[handledRoute] != nil
	})

	if existingRouteIndex > -1 {
		adapterInstance.Trie.Insert(routeWithParams, existingRouteIndex)
		adapterInstance.array[existingRouteIndex] = map[string]*routerData{
			handledRoute: routerDataInstance,
		}
	} else {
		adapterInstance.Trie.Insert(routeWithParams, len(adapterInstance.array))
		adapterInstance.array = append(adapterInstance.array, map[string]*routerData{
			handledRoute: routerDataInstance,
		})
	}

	return adapterInstance
}

func (adapterInstance *adapter) find(route string) (bool, string, *routerData) {
	handledRoute := handlePath(route)
	shadowOfRouterData := new(routerData)
	_, shadowOfRouterData.Params = context.NewParam("")
	shadowOfRouterData.Handlers = &[]context.Handler{}

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
