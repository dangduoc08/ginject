package express

import "strings"

type trie struct {
	node          map[string]*trie
	params        map[string]string
	httpMethod    map[string]bool
	handleRequest func(req *Request, res ResponseExtender)
	isEnd         bool
}

// Insert route path and its data to tree
func (t *trie) insert(word, method string, handleRequest func(req *Request, res ResponseExtender)) {
	if handleRequest == nil {
		panic("http: nil handler")
	}
	var lastIndex int = len(word) - 1
	var prefixParam string = ":"
	for currentIndex, runeStr := range word {
		var str string = string(runeStr)

		// If key haven't existed in map
		// create new one
		if t.node[str] == nil {
			var newTrie *trie = new(trie)
			newTrie.node = make(map[string]*trie)
			t.node[str] = newTrie
		}

		// Pass params and method to node has key = ":"
		// to easy to access params
		if str == prefixParam {
			if t.params == nil {
				params := make(map[string]string)
				t.params = params
			}
			// Handle case has many params
			// get param from first index to slash index
			var remainStr string = word[currentIndex+1:]
			var slashIndex int = strings.Index(remainStr, "/")

			if slashIndex > -1 {
				t.params[method] = remainStr[:slashIndex]
			} else {
				t.params[method] = remainStr
			}
		}

		// When loop runs to last index
		// flag isEnd will set true
		// add http method to map
		// add handle request function to map
		if currentIndex == lastIndex {
			t.node[str].isEnd = true
			t.node[str].handleRequest = handleRequest
			// If http method map didnt created
			// create new one
			if t.node[str].httpMethod == nil {
				t.node[str].httpMethod = make(map[string]bool)
			}
			t.node[str].httpMethod[method] = true
		}
		t = t.node[str]
	}
}

// Check client send URL if match in tree
func (t *trie) match(word, method string, params *map[string]string) (bool, func(req *Request, res ResponseExtender)) {
	var lastIndex int = len(word) - 1
	var remainStr string
	var prefixParam string = ":"
	var matched bool
	var handleRequest func(req *Request, res ResponseExtender)

	// Remove "/" at last index in URL
	if word != "/" && string(word[lastIndex]) == "/" {
		word = word[0:lastIndex]
		lastIndex--
	}

	// #CASE_1 URL with no params
	for currentIndex, runeStr := range word {
		var str string = string(runeStr)
		if t.node[str] != nil {

			// If match whole word (loop no break and isEnd = true)
			// http method matched
			// return handle request function is matched with route which client sent
			if currentIndex == lastIndex && t.node[str].isEnd && t.node[str].httpMethod[method] {
				matched = true
				handleRequest = t.node[str].handleRequest
			}
			t = t.node[str]
		} else {
			remainStr = word[currentIndex:]
			break
		}
	}

	// If #CASE_1 didn't matched
	// check URL variables
	if !matched && t.node[prefixParam] != nil && t.params[method] != "" {
		var paramVal string

		// A param consider from ":" to first "/"
		// after get param, remain string will replace params with ":<key_params>"
		// then run recursively with remain till remain string matched or unmatched both case
		var slashIndex int = strings.Index(remainStr, "/")

		// If slash index > -1 mean URL maybe have more than 1 params
		if slashIndex > -1 {
			paramVal = remainStr[0:slashIndex]
			remainStr = prefixParam + t.params[method] + remainStr[slashIndex:]
		} else {
			paramVal = remainStr[0:]
			remainStr = prefixParam + t.params[method]
		}
		(*params)[t.params[method]] = paramVal
		matched, handleRequest = t.match(remainStr, method, params)
	}

	return matched, handleRequest
}
