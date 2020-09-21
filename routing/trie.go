package routing

type trie struct {
	node  map[string]*trie
	isEnd bool
}

func new() *trie {
	return &trie{
		node:  make(map[string]*trie),
		isEnd: false,
	}
}

func (tr *trie) insert(chars string) *trie {
	l := len(chars) - 1
	shadowTrie := tr

	for i, rune := range chars {
		char := string(rune)
		isCharExistInNode := shadowTrie.node[char] != nil

		if !isCharExistInNode {
			shadowTrie.node[char] = new()
			if i == l {
				shadowTrie.node[char].isEnd = true
			}
		}
		shadowTrie = shadowTrie.node[char]
	}

	return tr
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

func (tr *trie) search(chars string) bool {
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
