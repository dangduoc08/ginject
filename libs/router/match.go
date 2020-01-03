package router

import "strings"

func (t *tree) match(word, method string, params *map[string]string) (bool, func(req *Request, res Response)) {
	var lastIndex int = len(word) - 1
	var remainStr string
	var prefixParam string = ":"
	var matched bool
	var handleRequest func(req *Request, res Response)

	// Remove "/" at last index in URL
	if string(word[lastIndex]) == "/" {
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
