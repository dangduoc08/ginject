package routing

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/gooh/core"
	dataStructure "github.com/dangduoc08/gooh/data-structure"
)

type routerData struct {
	Handlers *[]core.Handler
	Vars     *core.Var[interface{}]
}

type IRoutable interface {
	NewRouter() *IRoutable
	Add(path string) *IRoutable
	Match(path string) *IRoutable
}

type Router struct {
	*dataStructure.Trie
	array []map[string]*routerData
}

func NewRouter() *Router {
	tr := dataStructure.NewTrie()

	return &Router{
		Trie:  tr,
		array: []map[string]*routerData{},
	}
}

func (r *Router) Add(route string, handlers ...core.Handler) *Router {
	trieAdapter := adapter{
		r,
	}
	trieAdapter.insert(route, handlers...)

	return r
}

func (r *Router) Match(route string) (bool, string, *routerData) {
	trieAdapter := adapter{
		r,
	}

	return trieAdapter.find(route)
}

func (r *Router) Group(route string, subRs ...*Router) *Router {
	if route == "" {
		route = dataStructure.SLASH
	}
	route = dataStructure.RemoveAtEnd(route, dataStructure.SLASH)
	for _, subR := range subRs {
		trieAdapter := adapter{
			r,
		}
		for _, rdMap := range subR.array {
			for subRoute, rd := range rdMap {
				trieAdapter.insert(route+subRoute, *rd.Handlers...)
			}
		}
	}

	return r
}

func (r *Router) Use(args ...interface{}) {
	var route string
	for i, arg := range args {
		switch arg.(type) {
		case string:
			if i == 0 {
				route = handleRoute(arg.(string))
				matchedMap := dataStructure.Find(r.array, func(m map[string]*routerData, index int, arr []map[string]*routerData) bool {
					for k := range m {
						return k == route
					}
					return false
				})
				fmt.Println(matchedMap)
			}

		case core.Handler:
			fmt.Printf("heheh %T\n", arg)
		}

	}

}

func (r *Router) genTrieMap(c string) map[string]interface{} {
	tr := r.Trie
	params := []string{}
	handlers := []interface{}{}

	if tr.Index > -1 {
		var data *routerData
		for _, routerData := range r.array[tr.Index] {
			data = routerData
		}
		if data != nil {
			if data.Vars != nil {
				for k := range data.Vars.KeyValue {
					params = append(params, k)
				}
			}

			if data.Handlers != nil {
				for _, v := range *data.Handlers {
					handlerName := runtime.FuncForPC(reflect.ValueOf(v).Pointer()).Name()

					if handlerName == "" {
						handlers = append(handlers, nil)
						break
					} else {
						lastDotIndex := strings.LastIndex(handlerName, dataStructure.DOT)
						if lastDotIndex > -1 {
							handlerName = handlerName[lastDotIndex+1:]
						}
						handlers = append(handlers, handlerName)
					}
				}
			}
		}
	}

	nodes := []interface{}{}
	for k, v := range tr.Node {
		newR := Router{v, r.array}
		nodes = append(nodes, newR.genTrieMap(k))
	}

	visualizationMap := map[string]interface{}{
		"char":     c,
		"isEnd":    tr.IsEnd,
		"index":    tr.Index,
		"params":   params,
		"handlers": handlers,
		"nodes":    nodes,
	}

	return visualizationMap
}

func (r *Router) visualize() ([]byte, error) {
	vM := r.genTrieMap("root")
	return json.Marshal(vM)
}
