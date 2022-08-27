package routing

import (
	"github.com/dangduoc08/go-go/core"
	"github.com/dangduoc08/go-go/helper"
)

type node map[string]*trie

type trie struct {
	node  node
	isEnd bool
	index int
}

func newTrie() *trie {
	return &trie{
		node:  make(node),
		isEnd: false,
		index: -1,
	}
}

func (tr *trie) len() uint {
	var counter uint = 0

	for k, v := range tr.node {
		if k != "" {
			counter += 1
			if v != nil {
				counter += v.len()
			}
		}
	}

	return counter
}

func (tr *trie) insert(chars string, index int) *trie {
	l := len(chars) - 1
	shadowTrie := tr

	for i, rune := range chars {
		char := string(rune)
		isCharExistInNode := shadowTrie.node[char] != nil

		if !isCharExistInNode {
			shadowTrie.node[char] = newTrie()
			if i == l {
				shadowTrie.node[char].isEnd = true
				shadowTrie.node[char].index = index
			}
		}
		shadowTrie = shadowTrie.node[char]
	}

	return tr
}

func (tr *trie) find(chars string) (bool, int, []string) {
	l := len(chars)
	isFound := false
	isHasParam := false
	isHasWildcard := false
	var index int
	varValues := make([]string, 0)
	varValue := ""
	shadowTrie := tr

	for i, rune := range chars {
		char := string(rune)

		if char == helper.SLASH {
			isHasParam = false
			isHasWildcard = false
			if varValue != "" {
				varValues = append(varValues, varValue)
				varValue = ""
			}
		}

		if shadowTrie.node[char] == nil {

			// Handle routes have params
			// param have higher priority than wildcard
			if shadowTrie.node[core.VAR_SYMBOL] != nil {
				shadowTrie = shadowTrie.node[core.VAR_SYMBOL]
				isHasParam = true
			}

			if isHasParam {
				varValue += char
				continue
			}

			if shadowTrie.node[helper.WILDCARD] != nil {
				shadowTrie = shadowTrie.node[helper.WILDCARD]
				isHasWildcard = true
			}

			if isHasWildcard {
				continue
			}

			break
		}

		if i == l-1 {
			isFound = shadowTrie.node[char].isEnd
			index = shadowTrie.node[char].index
			break
		}

		shadowTrie = shadowTrie.node[char]
	}

	return isFound, index, varValues
}
