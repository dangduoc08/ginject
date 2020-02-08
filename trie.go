package express

import (
	"fmt"
	"strings"
)

type trie struct {
	node       map[string]*trie  // Node key is a word of route
	params     map[string]string // Route params from ":" to first "/"
	suffix     []string          // Route string from "*" to first "/"
	httpMethod map[string]bool   // Router method
	handlers   []Handler         // All middleware and endpoint handler
	isEnd      bool              // If end route, isEnd will set true
}

// Insert route path and its data to tree
func (t *trie) insert(word, method string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Error: nil handler")
	}
	var lastIndex int = len(word) - 1
	var prefixParam string = ":"
	var slash string = "/"
	var allPattern string = "*"

	// Remove "/" at last route
	if word != slash && string(word[lastIndex]) == slash {
		word = word[0:lastIndex]
		lastIndex--
	}

	// Add "/" at first route
	if string(word[0]) != slash {
		word = slash + word
		lastIndex++
	}

	for currentIndex, runeStr := range word {
		var str string = string(runeStr)

		// If key haven't existed in map
		// create new one
		if t.node[str] == nil {
			var newTrie *trie = new(trie)
			newTrie.node = make(map[string]*trie)
			t.node[str] = newTrie
		}

		// Pass URL route after "*"
		// to help we know whether has suffix or not
		if str == allPattern {
			var remainStr string = word[currentIndex+1:]
			var slashIndex int = strings.Index(remainStr, slash)
			var suffixKey string

			// Treating the string from "*" to "/" is suffix
			// therefore one router can have many suffix
			if slashIndex > -1 {
				suffixKey = remainStr[:slashIndex]
			} else {
				suffixKey = remainStr
			}

			if suffixKey != "" {
				t.suffix = append(t.suffix, suffixKey)
			}
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
			var slashIndex int = strings.Index(remainStr, slash)

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
			t.node[str].handlers = handlers
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
func (t *trie) match(word, method string, params *map[string]string) (bool, []Handler) {
	var lastIndex int = len(word) - 1
	var remainStr string
	var prefixParam string = ":"
	var slash string = "/"
	var allPattern string = "*"
	var matched bool
	var handlers []Handler

	// Remove "/" at last index in URL
	if word != slash && string(word[lastIndex]) == slash {
		word = word[0:lastIndex]
		lastIndex--
	}

	for currentIndex, runeStr := range word {
		var str string = string(runeStr)
		if t.node[str] != nil {

			// If match whole word (loop no break and isEnd = true)
			// http method matched
			// return handler functions is matched
			if currentIndex == lastIndex && t.node[str].isEnd && t.node[str].httpMethod[method] {
				matched = true
				handlers = t.node[str].handlers
			}
			t = t.node[str]

			// If route has "*"
			// placed at last index
			// it mean all URL after "*" will be matched
		} else if t.node[allPattern] != nil && t.node[allPattern].isEnd {
			matched = true
			handlers = t.node[allPattern].handlers

			// If route didn't matched
			// "*" not placed ai last index
			// keep the remain URL to check once more time with below recursive
		} else {
			remainStr = word[currentIndex:]
			break
		}
	}

	// With remain URL, divide into 2 cases
	// #CASE_1 router includes params
	// so remain URL start with ":"
	// check whether URL variables existed
	if !matched && t.node[prefixParam] != nil && t.params[method] != "" {
		var paramVal string

		// A param consider from ":" to first "/"
		// after get param, remain string will
		// replace params with ":<key_params>"
		// then run recursively with remain
		// till remain string matched or unmatched both case
		var slashIndex int = strings.Index(remainStr, slash)

		// If slash index > -1 mean URL maybe have more than 1 params
		if slashIndex > -1 {
			paramVal = remainStr[0:slashIndex]
			remainStr = prefixParam + t.params[method] + remainStr[slashIndex:]
		} else {
			paramVal = remainStr[0:]
			remainStr = prefixParam + t.params[method]
		}

		// Put param value to req.Params
		(*params)[t.params[method]] = paramVal
		matched, handlers = t.match(remainStr, method, params) // Recursive

		// #CASE_2 router includes "*"
		// check whether match all pattern existed
	} else if !matched && t.node[allPattern] != nil {

		// Suffix is an string array
		// check whether any suffix match with URL client send
		var suffixIndex int = -1
		for _, suffix := range t.suffix {
			suffixIndex = strings.Index(remainStr, suffix)
			if suffixIndex > -1 {
				break
			}
		}

		// 4 conditional statements below
		// solve 4 cases:
		// - "/before_*_after" => _after is suffix
		// - "/before_*/after" => no suffix but * not placed at last index
		// - "/before_*/:after" => no suffix but after * is a param
		// - "/before_*" => no suffix and * placed at last

		// After "*" has suffix
		if suffixIndex > -1 {
			remainStr = allPattern + remainStr[suffixIndex:]

			// After "*" has no suffix
		} else {
			var slashIndex int = strings.Index(remainStr, slash)

			// "*" not placed at last index
			if slashIndex > -1 {
				if t.node[allPattern].node[slash].node[prefixParam] != nil {
					remainStr = allPattern + remainStr[slashIndex+6:]
				} else {
					remainStr = allPattern + remainStr[slashIndex:]
				}

				// "*" placed at last index
			} else {
				remainStr = allPattern
			}
		}
		fmt.Println(remainStr)
		matched, handlers = t.match(remainStr, method, params) // Recursive
	}

	return matched, handlers
}
