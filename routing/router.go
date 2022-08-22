package routing

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/go-go/core"
	"github.com/dangduoc08/go-go/helper"
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
	*trie[*routerData]
	array []map[string]*routerData
}

func NewRouter() *Router {
	tr := newTrie[*routerData]()
	tr.data = new(routerData)

	return &Router{
		trie:  tr,
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

func (r *Router) Match(route string) (bool, *routerData) {
	trieAdapter := adapter{
		r,
	}

	return trieAdapter.find(route)
}

func (r *Router) Group(route string, subRs ...*Router) *Router {
	if route == helper.EMPTY {
		route = helper.SLASH
	}
	route = helper.RemoveAtEnd(route, helper.SLASH)
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

func (r *Router) genTrieMap(c string) map[string]interface{} {
	tr := r.trie
	params := []string{}
	handlers := []interface{}{}

	if tr.data != nil {
		if tr.data.Vars != nil {
			for k := range tr.data.Vars.KeyValue {
				params = append(params, k)
			}
		}

		if tr.data.Handlers != nil {
			for _, v := range *tr.data.Handlers {
				handlerName := runtime.FuncForPC(reflect.ValueOf(v).Pointer()).Name()

				if handlerName == helper.EMPTY {
					handlers = append(handlers, nil)
					break
				} else {
					lastDotIndex := strings.LastIndex(handlerName, helper.DOT)
					if lastDotIndex > -1 {
						handlerName = handlerName[lastDotIndex+1:]
					}
					handlers = append(handlers, handlerName)
				}
			}
		}
	}

	nodes := []interface{}{}
	for k, v := range tr.node {
		newR := Router{v, nil}
		nodes = append(nodes, newR.genTrieMap(k))
	}

	visualizationMap := map[string]interface{}{
		"char":     c,
		"isEnd":    tr.isEnd,
		"index":    tr.index,
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
