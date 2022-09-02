package ds

type Node map[string]*Trie

type Trie struct {
	Root  Node
	Index int
	isEnd bool
}

func NewTrie() *Trie {
	return &Trie{
		Root:  make(Node),
		isEnd: false,
		Index: -1,
	}
}

func (trieInstance *Trie) Len() uint {
	var counter uint = 0
	for word, node := range trieInstance.Root {
		if word != "" {
			counter += 1
			if node != nil {
				counter += node.Len()
			}
		}
	}

	return counter
}

func (trieInstance *Trie) Insert(words string, index int) *Trie {
	wordLength := len(words)
	shadowOfTrie := trieInstance

	for i, rune := range words {
		letter := string(rune)
		isLetterExistInRoot := shadowOfTrie.Root[letter] != nil

		if !isLetterExistInRoot {
			shadowOfTrie.Root[letter] = NewTrie()
		}

		if i == wordLength-1 {
			shadowOfTrie.Root[letter].isEnd = true
			shadowOfTrie.Root[letter].Index = index
		}
		shadowOfTrie = shadowOfTrie.Root[letter]
	}

	return trieInstance
}

func (trieInstance *Trie) Find(words string) (bool, int, []string) {
	wordLength := len(words)
	var Index int = -1
	paramValues := make([]string, 0)
	paramsValue := ""
	isEnd := false
	isHasParam := false
	isHasWildcard := false
	shadowOfTrie := trieInstance

	for i, rune := range words {
		letter := string(rune)

		if letter == SLASH {
			isHasParam = false
			isHasWildcard = false
			if paramsValue != "" {
				paramValues = append(paramValues, paramsValue)
				paramsValue = ""
			}
		}

		if shadowOfTrie.Root[letter] == nil {

			// Handle routes have params
			// param have higher priority than wildcard
			if shadowOfTrie.Root[DOLLAR_SIGN] != nil {
				shadowOfTrie = shadowOfTrie.Root[DOLLAR_SIGN]
				isHasParam = true
			}

			if isHasParam {
				paramsValue += letter
				continue
			}

			if shadowOfTrie.Root[WILDCARD] != nil {
				shadowOfTrie = shadowOfTrie.Root[WILDCARD]
				isHasWildcard = true
			}

			if isHasWildcard {
				continue
			}

			break
		}

		if i == wordLength-1 {
			isEnd = shadowOfTrie.Root[letter].isEnd
			Index = shadowOfTrie.Root[letter].Index
			break
		}

		shadowOfTrie = shadowOfTrie.Root[letter]
	}

	return isEnd, Index, paramValues
}
