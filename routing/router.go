package routing

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/core"
	dataStructure "github.com/dangduoc08/gooh/data-structure"
)

type routerData struct {
	Handlers *[]core.Handler
	Params   *context.Param[interface{}]
}

type Router struct {
	*dataStructure.Trie
	array       []map[string]*routerData
	middlewares middleware
}

func NewRouter() *Router {
	trieInstance := dataStructure.NewTrie()
	middlewareInstance := newMiddleware()

	return &Router{
		Trie:        trieInstance,
		array:       []map[string]*routerData{},
		middlewares: middlewareInstance,
	}
}

func (routerInstance *Router) Add(route string, handlers ...core.Handler) *Router {
	routerAdapter := adapter{
		routerInstance,
	}
	routerAdapter.insert(route, handlers...)
	routerAdapter.serve(route, ADD)

	return routerInstance
}

func (routerInstance *Router) Match(route string) (bool, string, *routerData) {
	routerAdapter := adapter{
		routerInstance,
	}

	return routerAdapter.find(route)
}

func (routerInstance *Router) Group(prefixRoute string, subRouters ...*Router) *Router {
	if prefixRoute == "" {
		prefixRoute = dataStructure.SLASH
	}
	prefixRoute = dataStructure.RemoveAtEnd(prefixRoute, dataStructure.SLASH)
	for _, subRouter := range subRouters {
		routerAdapter := adapter{
			routerInstance,
		}
		for _, subRouterDataMappedByRoute := range subRouter.array {
			for subRoute, subRouterData := range subRouterDataMappedByRoute {
				routerAdapter.insert(prefixRoute+subRoute, *subRouterData.Handlers...)
			}
		}
	}

	return routerInstance
}

func (routerInstance *Router) Use(handlers ...core.Handler) *Router {
	routerInstance.middlewares.cache(dataStructure.WILDCARD, handlers...)
	routerAdapter := adapter{
		routerInstance,
	}
	routerAdapter.serve(dataStructure.WILDCARD, USE, handlers...)

	return routerInstance
}

func (routerInstance *Router) For(route string) func(handlers ...core.Handler) *Router {
	routerAdapter := adapter{
		routerInstance,
	}

	return func(handlers ...core.Handler) *Router {
		routerInstance.middlewares.cache(handleRoute(route), handlers...)
		routerAdapter.serve(route, USE, handlers...)

		return routerInstance
	}
}

func (routerInstance *Router) genTrieMap(word string) map[string]interface{} {
	routerTrie := routerInstance.Trie
	params := []string{}
	handlers := []interface{}{}

	if routerTrie.Index > -1 {
		var routerData *routerData
		for _, routerDataPt := range routerInstance.array[routerTrie.Index] {
			routerData = routerDataPt
		}
		if routerData != nil {
			if routerData.Params != nil {
				params = append(params, routerData.Params.Keys()...)
			}

			if routerData.Handlers != nil {
				for _, handler := range *routerData.Handlers {
					handlerFuncName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

					if handlerFuncName == "" {
						handlers = append(handlers, nil)
						break
					} else {
						lastDotIndex := strings.LastIndex(handlerFuncName, dataStructure.DOT)
						if lastDotIndex > -1 {
							handlerFuncName = handlerFuncName[lastDotIndex+1:]
						}
						handlers = append(handlers, handlerFuncName)
					}
				}
			}
		}
	}

	nodes := []interface{}{}
	for nextWord, nextNode := range routerTrie.Root {
		newRouterInstance := Router{nextNode, routerInstance.array, newMiddleware()}
		nodes = append(nodes, newRouterInstance.genTrieMap(nextWord))
	}

	visualizationMap := map[string]interface{}{
		"word":     word,
		"index":    routerTrie.Index,
		"params":   params,
		"handlers": handlers,
		"nodes":    nodes,
	}

	return visualizationMap
}

func (routerInstance *Router) visualize() ([]byte, error) {
	visualizationMap := routerInstance.genTrieMap("root")
	return json.Marshal(visualizationMap)
}
