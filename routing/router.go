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
	array []map[string]*routerData
}

func NewRouter() *Router {
	trieInstance := dataStructure.NewTrie()

	return &Router{
		Trie:  trieInstance,
		array: []map[string]*routerData{},
	}
}

func (routerInstance *Router) Add(route string, handlers ...core.Handler) *Router {
	routerAdapter := adapter{
		routerInstance,
	}
	routerAdapter.insert(route, handlers...)

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

// func (r *Router) Use(args ...interface{}) {
// 	var route string
// 	for i, arg := range args {
// 		switch arg.(type) {
// 		case string:
// 			if i == 0 {
// 				route = handleRoute(arg.(string))
// 				matchedMap := dataStructure.Find(r.array, func(m map[string]*routerData, index int, arr []map[string]*routerData) bool {
// 					for k := range m {
// 						return k == route
// 					}
// 					return false
// 				})
// 				fmt.Println(matchedMap)
// 			}

// 		case core.Handler:
// 			fmt.Printf("heheh %T\n", arg)
// 		}

// 	}

// }

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
		newRouterInstance := Router{nextNode, routerInstance.array}
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
