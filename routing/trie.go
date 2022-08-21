package routing

import (
	"github.com/dangduoc08/go-go/core"
	"github.com/dangduoc08/go-go/helper"
)

type node[T any] map[string]*trie[T]

type trie[T any] struct {
	node  node[T]
	isEnd bool
	data  T
}

func newTrie[T any]() *trie[T] {
	return &trie[T]{
		node:  make(node[T]),
		isEnd: false,
	}
}

func (tr *trie[T]) len() uint {
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

func (tr *trie[T]) insert(chars string, data T) *trie[T] {
	l := len(chars) - 1
	shadowTrie := tr

	for i, rune := range chars {
		char := string(rune)
		isCharExistInNode := shadowTrie.node[char] != nil

		if !isCharExistInNode {
			shadowTrie.node[char] = newTrie[T]()
			if i == l {
				shadowTrie.node[char].isEnd = true
				shadowTrie.node[char].data = data
			}
		}
		shadowTrie = shadowTrie.node[char]
	}

	return tr
}

func (tr *trie[T]) find(chars string) (bool, []string, T) {
	l := len(chars)
	isFound := false
	isHasParam := false
	isHasWildcard := false
	varValues := make([]string, 0)
	varValue := helper.EMPTY
	shadowTrie := tr
	var rD T

	for i, rune := range chars {
		char := string(rune)

		if char == helper.SLASH {
			isHasParam = false
			isHasWildcard = false
			if varValue != helper.EMPTY {
				varValues = append(varValues, varValue)
				varValue = helper.EMPTY
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
			rD = shadowTrie.node[char].data
			break
		}

		shadowTrie = shadowTrie.node[char]
	}

	return isFound, varValues, rD
}
