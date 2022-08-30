package data_structure

type Node map[string]*Trie

type Trie struct {
	Node  Node
	IsEnd bool
	Index int
}

func NewTrie() *Trie {
	return &Trie{
		Node:  make(Node),
		IsEnd: false,
		Index: -1,
	}
}

func (tr *Trie) Len() uint {
	var counter uint = 0

	for k, v := range tr.Node {
		if k != "" {
			counter += 1
			if v != nil {
				counter += v.Len()
			}
		}
	}

	return counter
}

func (tr *Trie) Insert(chars string, Index int) *Trie {
	l := len(chars) - 1
	shadowTrie := tr

	for i, rune := range chars {
		char := string(rune)
		isCharExistInNode := shadowTrie.Node[char] != nil

		if !isCharExistInNode {
			shadowTrie.Node[char] = NewTrie()
			if i == l {
				shadowTrie.Node[char].IsEnd = true
				shadowTrie.Node[char].Index = Index
			}
		}
		shadowTrie = shadowTrie.Node[char]
	}

	return tr
}

func (tr *Trie) Find(chars string) (bool, int, []string) {
	l := len(chars)
	isFound := false
	isHasParam := false
	isHasWildcard := false
	var Index int
	varValues := make([]string, 0)
	varValue := ""
	shadowTrie := tr

	for i, rune := range chars {
		char := string(rune)

		if char == SLASH {
			isHasParam = false
			isHasWildcard = false
			if varValue != "" {
				varValues = append(varValues, varValue)
				varValue = ""
			}
		}

		if shadowTrie.Node[char] == nil {

			// Handle routes have params
			// param have higher priority than wildcard
			if shadowTrie.Node[DOLLAR_SIGN] != nil {
				shadowTrie = shadowTrie.Node[DOLLAR_SIGN]
				isHasParam = true
			}

			if isHasParam {
				varValue += char
				continue
			}

			if shadowTrie.Node[WILDCARD] != nil {
				shadowTrie = shadowTrie.Node[WILDCARD]
				isHasWildcard = true
			}

			if isHasWildcard {
				continue
			}

			break
		}

		if i == l-1 {
			isFound = shadowTrie.Node[char].IsEnd
			Index = shadowTrie.Node[char].Index
			break
		}

		shadowTrie = shadowTrie.Node[char]
	}

	return isFound, Index, varValues
}
