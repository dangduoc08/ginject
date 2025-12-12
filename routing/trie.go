package routing

import (
	"encoding/json"
	"reflect"
	"runtime"
	"strings"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/utils"
)

type (
	Node   map[string]*Trie
	ScanFn func(string, *Trie)
)

type Trie struct {
	Children  Node
	Handlers  []ctx.Handler
	ParamKeys map[string][]int
	Index     int
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

func (tr *Trie) insert(path string, sep byte, index int, paramKeys map[string][]int, handlers []ctx.Handler) *Trie {
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
			node.Children[seg].Handlers = handlers
		}
		node = node.Children[seg]
	}

	return tr
}

func (tr *Trie) find(path, method, version string, sep byte) (int, map[string][]int, []string, []ctx.Handler) {
	node := tr
	var matchedNode *Trie
	var lastWildcardNode *Trie
	start := strings.IndexByte(path, sep)

	i := -1
	paramKeys := make(map[string][]int)
	paramVals := make([]string, 0)
	handlers := []ctx.Handler{}
	methodPattern := fromMethodtoPattern(method)
	versionPattern := fromVersiontoPattern(version)

	for seg, next := utils.StrSegment(path, sep, start); next > -1; seg, next = utils.StrSegment(path, sep, next) {
		if node.Children[seg] == nil {

			// Handle segs have paramVals
			// param have higher priority than wildcard
			// pushed /lv1/123 => /lv/{id}
			if node.Children["$"] != nil {

				// handle case param and wildcard on same position
				// then cannot fallback to wildcard
				// due to trie already be traversed
				// we will store temp node and return if no route matched
				lastWildcardNode = getLastWildcardNode(node, versionPattern, methodPattern)

				// pushed /lv1 => /lv/{id}
				// but still matched
				// due to [GET] will be treated as param value
				// can match due to line 172
				// this line prevent this
				if seg == methodPattern && next == len(path)-1 {
					break
				}

				node = node.Children["$"]
				paramVals = append(paramVals, seg)
			} else if node.Children["*"] != nil {
				lastWildcardNode = getLastWildcardNode(node, versionPattern, methodPattern)
				node = node.Children["*"]
			} else {
				isNotMatchAnythings := true

				// check prefix*suffix case
				// useful when want to use route like:
				// *.html, filename.*
				// limitation:
				// if we pushed /lv1/* and /lv1/*/*.html
				// then /lv1/* will match
				for route := range node.Children {
					if matchWildcard(seg, route) {
						node = node.Children[route]
						isNotMatchAnythings = false
						break
					}
				}

				// if not matched any route
				// but has last wildcard node
				// then fallback to lastWildcardNode
				// jump to line 185
				// if not break in this conditions
				// pushed /lv1/{id} and /lv1/*
				// request /lv1/foo/bar won't match /lv1/*
				// instead it's matched /lv1/{id}
				if isNotMatchAnythings ||
					(isNotMatchAnythings && lastWildcardNode != nil) {
					break
				}
			}
		} else {

			// handle case static path and wildcard on same position
			// then cannot fallback to wildcard
			// due to trie already be traversed
			// we will store temp node and return if no route matched
			lastWildcardNode = getLastWildcardNode(node, versionPattern, methodPattern)
			node = node.Children[seg]
		}

		if next == len(path)-1 {
			matchedNode = node

			// if not matched any route
			// but has last wildcard node
			// then fallback to lastWildcardNode
			if matchedNode.Index < 0 && lastWildcardNode != nil {
				matchedNode = lastWildcardNode
			}

			i = matchedNode.Index
			paramKeys = matchedNode.ParamKeys
			handlers = matchedNode.Handlers
			break
		}

		continue
	}

	if i < 0 && lastWildcardNode != nil {
		matchedNode = lastWildcardNode
		i = matchedNode.Index
		paramKeys = matchedNode.ParamKeys
		handlers = matchedNode.Handlers
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
