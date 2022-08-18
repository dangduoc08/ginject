package routing

type trie[T any] struct {
	node       map[string]*trie[T]
	isEnd      bool
	routerData T
}

func newTrie[T any]() *trie[T] {
	return &trie[T]{
		node:  make(map[string]*trie[T]),
		isEnd: false,
	}
}

func (tr *trie[T]) insert(chars string, routerData T) *trie[T] {
	l := len(chars) - 1
	shadowTrie := tr

	for i, rune := range chars {
		char := string(rune)
		isCharExistInNode := shadowTrie.node[char] != nil

		if !isCharExistInNode {
			shadowTrie.node[char] = newTrie[T]()
			if i == l {
				shadowTrie.node[char].isEnd = true
				shadowTrie.node[char].routerData = routerData
			}
		}
		shadowTrie = shadowTrie.node[char]
	}

	return tr
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

func (tr *trie[T]) search(chars string) bool {
	l := len(chars) - 1
	shadowTrie := tr
	isFound := false

	for i, rune := range chars {
		char := string(rune)

		if shadowTrie.node[char] == nil {
			break
		}

		if i == l {
			isFound = shadowTrie.node[char].isEnd
			break
		}

		shadowTrie = shadowTrie.node[char]
	}

	return isFound
}
