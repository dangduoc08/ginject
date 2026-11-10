package ds

import (
	"strings"

	"github.com/dangduoc08/ginject/internal/str"
)

type Node map[string]*Trie

type Trie struct {
	IsEnd        bool
	Raw          string
	Children     Node
	param        *Trie
	wildcard     *Trie
	globChildren []string
}

const paramValsInitialCap = 4

func NewTrie() *Trie {
	return &Trie{
		Children: make(Node),
	}
}

func (tr *Trie) Insert(raw, insertedStr string, sep byte) *Trie {
	node := tr
	start := strings.IndexByte(insertedStr, sep)

	for seg, next := str.Segment(insertedStr, sep, start); next > -1; seg, next = str.Segment(insertedStr, sep, next) {
		child := node.Children[seg]
		if child == nil {
			child = NewTrie()
			node.Children[seg] = child
			switch {
			case seg == "$":
				node.param = child
			case seg == "*":
				node.wildcard = child
			case strings.Contains(seg, "*"):
				node.globChildren = append(node.globChildren, seg)
			}
		}

		if next == len(insertedStr)-1 {
			child.IsEnd = true
			child.Raw = raw
		}
		node = child
	}

	return tr
}

func (tr *Trie) Find(path string, sep byte) (string, string, []string) {
	matchedRaw, wildcardRaw, paramVals := tr.findGreedy(path, sep)
	if matchedRaw != "" || wildcardRaw != "" {
		return matchedRaw, wildcardRaw, paramVals
	}

	var backtrackParams []string
	if raw, ok := tr.search(path, sep, strings.IndexByte(path, sep), &backtrackParams); ok {
		if len(backtrackParams) == 0 {
			backtrackParams = nil
		}
		return raw, "", backtrackParams
	}
	return matchedRaw, wildcardRaw, paramVals
}

func (tr *Trie) findGreedy(path string, sep byte) (string, string, []string) {
	node := tr
	var matchedNode *Trie
	var wildcardNode *Trie
	start := strings.IndexByte(path, sep)

	var paramVals []string

	for seg, next := str.Segment(path, sep, start); next > -1; seg, next = str.Segment(path, sep, next) {
		if w := node.wildcard; w != nil && w.IsEnd {
			wildcardNode = w
		}

		if exactNode := node.exactChild(seg); exactNode != nil {
			node = exactNode
		} else if node.param != nil {
			node = node.param
			if paramVals == nil {
				paramVals = make([]string, 0, paramValsInitialCap)
			}
			paramVals = append(paramVals, seg)
		} else if node.wildcard != nil {
			node = node.wildcard
		} else {
			matched := false
			for _, route := range node.globChildren {
				if matchWildcard(seg, route) {
					node = node.Children[route]
					matched = true
					break
				}
			}
			if !matched {
				break
			}
		}

		if next == len(path)-1 {
			matchedNode = node
			if w := node.wildcard; w != nil && w.IsEnd {
				wildcardNode = w
			}
			break
		}
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

func (tr *Trie) search(path string, sep byte, pos int, params *[]string) (string, bool) {
	seg, next := str.Segment(path, sep, pos)
	if next < 0 {
		return "", false
	}
	isLast := next == len(path)-1

	descend := func(child *Trie, isParam bool) (string, bool) {
		if child == nil {
			return "", false
		}
		if isParam {
			*params = append(*params, seg)
		}
		if isLast {
			if child.IsEnd {
				return child.Raw, true
			}
		} else if raw, ok := child.search(path, sep, next, params); ok {
			return raw, true
		}
		if isParam {
			*params = (*params)[:len(*params)-1]
		}
		return "", false
	}

	if raw, ok := descend(tr.exactChild(seg), false); ok {
		return raw, true
	}
	if raw, ok := descend(tr.param, true); ok {
		return raw, true
	}
	if raw, ok := descend(tr.wildcard, false); ok {
		return raw, true
	}
	for _, route := range tr.globChildren {
		if matchWildcard(seg, route) {
			if raw, ok := descend(tr.Children[route], false); ok {
				return raw, true
			}
		}
	}
	return "", false
}

func (tr *Trie) exactChild(seg string) *Trie {
	if seg == "$" || strings.IndexByte(seg, '*') >= 0 {
		return nil
	}
	return tr.Children[seg]
}

func matchWildcard(segment, route string) bool {
	prefix, rest, found := strings.Cut(route, "*")
	if !found {
		return segment == route
	}

	if prefix != "" {
		if len(segment) < len(prefix) || segment[:len(prefix)] != prefix {
			return false
		}
		segment = segment[len(prefix):]
	}

	for {
		mid, next, found := strings.Cut(rest, "*")
		if !found {
			if mid == "" {
				return true
			}
			return len(segment) >= len(mid) && segment[len(segment)-len(mid):] == mid
		}
		rest = next

		if mid == "" {
			continue
		}

		idx := strings.Index(segment, mid)
		if idx < 0 {
			return false
		}
		segment = segment[idx+len(mid):]
	}
}
