package ds

import (
	"encoding/json"
	"strings"

	"github.com/dangduoc08/ginject/internal/str"
)

type Node map[string]*Trie

type Trie struct {
	Index    int
	Raw      string
	Children Node
}

const paramValsInitialCap = 4

func NewTrie() *Trie {
	return &Trie{
		Children: make(Node),
		Index:    -1,
	}
}

func (tr *Trie) Len() int {
	counter := 0
	for route, node := range tr.Children {
		if route != "" {
			counter++
			if node != nil {
				counter += node.Len()
			}
		}
	}

	return counter
}

func (tr *Trie) Insert(raw, insertedStr string, sep byte, index int) *Trie {
	node := tr
	start := strings.IndexByte(insertedStr, sep)

	for seg, next := str.Segment(insertedStr, sep, start); next > -1; seg, next = str.Segment(insertedStr, sep, next) {
		if node.Children[seg] == nil {
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

func (tr *Trie) Find(path string, sep byte) (int, string, int, string, []string) {
	node := tr
	var matchedNode *Trie
	var wildcardNode *Trie
	start := strings.IndexByte(path, sep)

	var paramVals []string

	for seg, next := str.Segment(path, sep, start); next > -1; seg, next = str.Segment(path, sep, next) {
		exactNode := node.Children[seg]
		if exactNode == nil {

			if node.Children["$"] != nil {

				if w := node.Children["*"]; w != nil && w.Index > -1 {
					wildcardNode = w
				}

				node = node.Children["$"]
				if paramVals == nil {
					paramVals = make([]string, 0, paramValsInitialCap)
				}
				paramVals = append(paramVals, seg)
			} else if node.Children["*"] != nil {
				if w := node.Children["*"]; w.Index > -1 {
					wildcardNode = w
				}
				node = node.Children["*"]
			} else {
				isNotMatchAnythings := true

				for route := range node.Children {
					if matchWildcard(seg, route) {
						node = node.Children[route]
						isNotMatchAnythings = false
						break
					}
				}

				if isNotMatchAnythings {
					break
				}
			}
		} else {

			if w := node.Children["*"]; w != nil && w.Index > -1 {
				wildcardNode = w
			}
			node = exactNode
		}

		if next == len(path)-1 {
			matchedNode = node

			if w := node.Children["*"]; w != nil && w.Index > -1 {
				wildcardNode = w
			}
			break
		}

		continue
	}

	matchedIndex := -1
	matchedRaw := ""
	if matchedNode != nil {
		matchedIndex = matchedNode.Index
		matchedRaw = matchedNode.Raw
	}
	wildcardIndex := -1
	wildcardRaw := ""
	if wildcardNode != nil {
		wildcardIndex = wildcardNode.Index
		wildcardRaw = wildcardNode.Raw
	}

	return matchedIndex, matchedRaw, wildcardIndex, wildcardRaw, paramVals
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
