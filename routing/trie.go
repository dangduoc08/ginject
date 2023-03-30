package routing

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/gooh/ctx"
	"github.com/dangduoc08/gooh/utils"
)

type (
	Node   map[string]*Trie
	ScanFn func(*Trie)
)

type Trie struct {
	Children  Node
	Handlers  []ctx.Handler
	ParamKeys map[string][]int
	Index     int
}

type Trier interface {
	len() int
	insert(string, byte, int, map[string][]int, []ctx.Handler) Trier
	find(string, byte) (int, map[string][]int, []string, []ctx.Handler)
	scan(cb ScanFn)
	ToJSON() (string, error)
}

func NewTrie() *Trie {
	return &Trie{
		Children: make(Node),
		Index:    -1,
	}
}

func (tr *Trie) len() int {
	counter := 0
	for route, node := range tr.Children {
		if route != "" {
			counter += 1
			if node != nil {
				counter += node.len()
			}
		}
	}

	return counter
}

func (tr *Trie) insert(path string, sep byte, index int, paramKeys map[string][]int, handlers []ctx.Handler) Trier {
	node := tr
	start := strings.IndexByte(path, sep)

	for seg, next := utils.StrSegment(path, sep, start); next > -1; seg, next = utils.StrSegment(path, sep, next) {
		isExist := node.Children[seg] != nil

		if !isExist {
			node.Children[seg] = NewTrie()
		}

		if next == len(path)-1 {
			node.Children[seg].Index = index
			node.Children[seg].ParamKeys = paramKeys
			node.Children[seg].Handlers = append(node.Children[seg].Handlers, handlers...)
		}
		node = node.Children[seg]
	}

	return tr
}

func (tr *Trie) find(path string, sep byte) (int, map[string][]int, []string, []ctx.Handler) {
	node := tr
	start := strings.IndexByte(path, sep)

	i := -1
	paramKeys := make(map[string][]int)
	paramVals := make([]string, 0)
	handlers := []ctx.Handler{}

	for seg, next := utils.StrSegment(path, sep, start); next > -1; seg, next = utils.StrSegment(path, sep, next) {
		if node.Children[seg] == nil {

			// Handle segs have paramVals
			// param have higher priority than wildcard
			if node.Children["$"] != nil {
				node = node.Children["$"]
				paramVals = append(paramVals, seg)
			} else if node.Children["*"] != nil {
				node = node.Children["*"]
			} else {
				for route := range node.Children {
					if matchWildcard(seg, route) {
						node = node.Children[route]
						break
					}
				}
			}
		} else {
			node = node.Children[seg]
		}

		if next == len(path)-1 {
			i = node.Index
			paramKeys = node.ParamKeys
			handlers = node.Handlers
			break
		}

		continue
	}

	return i, paramKeys, paramVals, handlers
}

func (tr *Trie) ToJSON() (string, error) {
	nodeMap := tr.genTrieMap("")
	b, err := json.Marshal(nodeMap)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func (tr *Trie) genTrieMap(path string) map[string]any {
	nodeMap := map[string]any{
		"children": []map[string]any{},
	}
	if path != "" {
		nodeMap["path"] = path
	}

	for route, node := range tr.Children {
		if route != "" {
			if node.Children != nil {
				trieMap := node.genTrieMap(route)
				trieMap["index"] = node.Index
				trieMap["params"] = node.ParamKeys

				if len(node.Handlers) > 0 {
					handlers := []any{}
					for _, handler := range node.Handlers {
						fnName := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

						if fnName == "" {
							handlers = append(handlers, nil)
							break
						} else {
							lastDotIndex := strings.LastIndex(fnName, ".")
							if lastDotIndex > -1 {
								fnName = fnName[lastDotIndex+1:]
							}
							handlers = append(handlers, fnName)
						}
					}
					trieMap["handlers"] = handlers
				}

				nodeMap["children"] = append(nodeMap["children"].([]map[string]any), trieMap)
			}
		}
	}

	return nodeMap

}

func (tr *Trie) scan(cb ScanFn) {
	for _, node := range tr.Children {
		if node.Index > -1 {
			cb(node)
		}
		node.scan(cb)
	}
}
