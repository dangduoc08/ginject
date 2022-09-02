package routing

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/gooh/ds"
)

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
						lastDotIndex := strings.LastIndex(handlerFuncName, ds.DOT)
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

// Support to debug router easier
func (routerInstance *Router) visualize() ([]byte, error) {
	visualizationMap := routerInstance.genTrieMap("root")
	return json.Marshal(visualizationMap)
}
