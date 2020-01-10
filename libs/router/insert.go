package router

import "strings"

// Insert route path and its data to tree
func (t *tree) insert(word, method string, handleRequest func(req *Request, res ResponseExtender)) {
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
			var newTree *tree = new(tree)
			newTree.node = make(map[string]*tree)
			t.node[str] = newTree
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
