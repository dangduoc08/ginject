package ds

import (
	"encoding/json"
	"strings"

	"github.com/dangduoc08/ginject/internal/str"
)

type Node map[string]*Trie

type Trie struct {
	IsEnd    bool
	Raw      string
	Children Node
}

const paramValsInitialCap = 4

func NewTrie() *Trie {
	return &Trie{
		Children: make(Node),
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

func (tr *Trie) Insert(raw, insertedStr string, sep byte) *Trie {
	node := tr
	start := strings.IndexByte(insertedStr, sep)

	for seg, next := str.Segment(insertedStr, sep, start); next > -1; seg, next = str.Segment(insertedStr, sep, next) {
		if node.Children[seg] == nil {
			node.Children[seg] = NewTrie()
		}

		if next == len(insertedStr)-1 {
			node.Children[seg].IsEnd = true
			node.Children[seg].Raw = raw
		}
		node = node.Children[seg]
	}

	return tr
}

func (tr *Trie) Remove(removedStr string, sep byte) bool {
	type step struct {
		parent *Trie
		key    string
	}

	node := tr
	start := strings.IndexByte(removedStr, sep)
	var chain []step

	for seg, next := str.Segment(removedStr, sep, start); next > -1; seg, next = str.Segment(removedStr, sep, next) {
		child := node.Children[seg]
		if child == nil {
			return false
		}
		chain = append(chain, step{parent: node, key: seg})
		node = child
	}

	if len(chain) == 0 || !node.IsEnd {
		return false
	}

	node.IsEnd = false
	node.Raw = ""

	for i := len(chain) - 1; i >= 0; i-- {
		s := chain[i]
		child := s.parent.Children[s.key]
		if child.IsEnd || len(child.Children) > 0 {
			break
		}
		delete(s.parent.Children, s.key)
	}

	return true
}

func (tr *Trie) Find(path string, sep byte, supportParams bool) (string, string, []string) {
	node := tr
	var matchedNode *Trie
	var wildcardNode *Trie
	start := strings.IndexByte(path, sep)

	var paramVals []string

	for seg, next := str.Segment(path, sep, start); next > -1; seg, next = str.Segment(path, sep, next) {
		exactNode := node.Children[seg]
		if exactNode == nil {

			if supportParams && node.Children["$"] != nil {

				if w := node.Children["*"]; w != nil && w.IsEnd {
					wildcardNode = w
				}

				node = node.Children["$"]
				if paramVals == nil {
					paramVals = make([]string, 0, paramValsInitialCap)
				}
				paramVals = append(paramVals, seg)
			} else if node.Children["*"] != nil {
				if w := node.Children["*"]; w.IsEnd {
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

			if w := node.Children["*"]; w != nil && w.IsEnd {
				wildcardNode = w
			}
			node = exactNode
		}

		if next == len(path)-1 {
			matchedNode = node

			if w := node.Children["*"]; w != nil && w.IsEnd {
				wildcardNode = w
			}
			break
		}

		continue
	}

	matchedRaw := ""
	if matchedNode != nil {
		matchedRaw = matchedNode.Raw
	}
	wildcardRaw := ""
	if wildcardNode != nil {
		wildcardRaw = wildcardNode.Raw
	}

	return matchedRaw, wildcardRaw, paramVals
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
				trieMap["isEnd"] = node.IsEnd

				nodeMap["children"] = append(nodeMap["children"].([]map[string]any), trieMap)
			}
		}
	}

	return nodeMap

}
