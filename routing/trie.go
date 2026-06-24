package routing

import (
	"encoding/json"
	"strings"

	"github.com/dangduoc08/ginject/internal/str"
)

type Node map[string]*Trie

type Trie struct {
	Children Node
	Index    int
	Raw      string
}

const paramValsInitialCap = 4

var methodPatternCache = func() map[string]string {
	m := make(map[string]string, len(HTTPMethods))
	for _, method := range HTTPMethods {
		m[method] = toPattern(method, "[", "]")
	}
	return m
}()

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
			counter++
			if node != nil {
				counter += node.len()
			}
		}
	}

	return counter
}

func (tr *Trie) insert(raw, insertedStr string, sep byte, index int) *Trie {
	node := tr
	start := strings.IndexByte(insertedStr, sep)

	for seg, next := str.Segment(insertedStr, sep, start); next > -1; seg, next = str.Segment(insertedStr, sep, next) {
		isExist := node.Children[seg] != nil

		if !isExist {
			node.Children[seg] = NewTrie()
		}

		if next == len(insertedStr)-1 {
			node.Children[seg].Index = index
			node.Children[seg].Raw = raw
		}
		node = node.Children[seg]
	}

	return tr
}

func (tr *Trie) find(path, method, version string, sep byte) (int, string, []string) {
	node := tr
	var matchedNode *Trie
	var wildcardNode *Trie
	start := strings.IndexByte(path, sep)

	i := -1
	raw := ""
	var paramVals []string
	versionPattern := "||"
	if version != "" {
		versionPattern = toPattern(version, "|", "|")
	}
	methodPattern, ok := methodPatternCache[method]
	if !ok {
		methodPattern = toPattern(method, "[", "]")
	}

	for seg, next := str.Segment(path, sep, start); next > -1; seg, next = str.Segment(path, sep, next) {
		exactNode := node.Children[seg]
		if exactNode == nil {

			// Handle segs have paramVals
			// param have higher priority than wildcard
			// pushed /lv1/123 => /lv/{id}
			if node.Children["$"] != nil {

				// handle case param and wildcard on same position
				// then cannot fallback to wildcard
				// due to trie already be traversed
				// we will store temp node and return if no route matched
				if node.Children["*"] != nil {
					if w := resolveWildcardRoute(node, versionPattern, methodPattern); w != nil {
						wildcardNode = w
					}
				}

				// pushed /lv1 => /lv/{id}
				// but still matched
				// due to [GET] will be treated as param value
				// can match due to line 172
				// this line prevent this
				if seg == methodPattern && next == len(path)-1 {
					break
				}

				node = node.Children["$"]
				if paramVals == nil {
					paramVals = make([]string, 0, paramValsInitialCap)
				}
				paramVals = append(paramVals, seg)
			} else if node.Children["*"] != nil {
				if w := resolveWildcardRoute(node, versionPattern, methodPattern); w != nil {
					wildcardNode = w
				}
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
				// then fallback to wildcardNode
				// jump to line 185
				// if not break in this conditions
				// pushed /lv1/{id} and /lv1/*
				// request /lv1/foo/bar won't match /lv1/*
				// instead it's matched /lv1/{id}
				if isNotMatchAnythings {
					break
				}
			}
		} else {

			// handle case static path and wildcard on same position
			// then cannot fallback to wildcard
			// due to trie already be traversed
			// we will store temp node and return if no route matched
			if node.Children["*"] != nil {
				if w := resolveWildcardRoute(node, versionPattern, methodPattern); w != nil {
					wildcardNode = w
				}
			}
			node = exactNode
		}

		if next == len(path)-1 {
			matchedNode = node

			// if not matched any route
			// but has last wildcard node
			// then fallback to wildcardNode
			if matchedNode.Index < 0 && wildcardNode != nil {
				matchedNode = wildcardNode
			}

			i = matchedNode.Index
			raw = matchedNode.Raw
			break
		}

		continue
	}

	if i < 0 && wildcardNode != nil {
		matchedNode = wildcardNode
		i = matchedNode.Index
		raw = matchedNode.Raw
	}

	return i, raw, paramVals
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

				nodeMap["children"] = append(nodeMap["children"].([]map[string]any), trieMap)
			}
		}
	}

	return nodeMap

}
