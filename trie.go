package gogo

import (
	"errors"
	"fmt"
	"strings"
)

type trie struct {
	node     map[string]*trie  // Node key is a word of route
	params   map[string]string // Route params from ":" to first "/"
	suffix   []string          // Route string from "*" to first "/"
	handlers []Handler         // All middleware and endpoint handler
	isEnd    bool              // If end route, isEnd will set true
}

const (
	param    string = ":"
	slash    string = "/"
	wildcard string = "*"
	empty    string = ""
)

func (t *trie) checkConflictWildcard(route string, currentIndex int) error {
	var nextIndex int = currentIndex + 1
	var remainRoute string = route[nextIndex:]
	var afterSlashWord string = string(remainRoute[0])
	var e error
	fmt.Printf("hihi")
	// #CASE 1: Insert absolute path first
	// then insert param or any
	var isInsertAbsolutePathFirst bool = afterSlashWord == param || afterSlashWord == wildcard

	// #CASE 2: Insert param or any first
	// then insert absolute path
	var isInsertParamOrWildcardFirst bool = t.node[slash].node[param] != nil || t.node[slash].node[wildcard] != nil

	if isInsertAbsolutePathFirst || isInsertParamOrWildcardFirst {
		var remainRouteSlashIndex int = strings.Index(remainRoute, slash)
		var conflictWord string

		if remainRouteSlashIndex > -1 {
			conflictWord = remainRoute[0:remainRouteSlashIndex]
		} else {
			conflictWord = remainRoute[0:]
		}

		// Generate error message
		var routeSlashIndex int = strings.Index(route, slash)
		var method string = route[0:routeSlashIndex]
		var pattern string = route[routeSlashIndex:]

		var message string = fmt.Sprintf(
			"wildcard '%s' in route %s('%s') conflicts with existing prefix in trie",
			conflictWord,
			method,
			pattern,
		)

		e = errors.New(message)
	}

	return e
}

// Insert route into trie
func (t *trie) insert(route, httpMethod string, handlers ...Handler) {
	if len(handlers) <= 0 {
		panic("Nil handler")
	}
	// fmt.Printf("[%v] %v\n", httpMethod, route)
	// Concat http method to route
	// to define what incoming request method is
	route = httpMethod + route

	var lastIndex int = len(route) - 1

	// If not match, matched handlers always return empty slice
	isExistedInTrie, matchedHandlers := t.match(route, httpMethod, make(map[string]string))

	// Route has existed in trie
	// append new handlers into existing handlers
	if isExistedInTrie && len(matchedHandlers) > 0 {
		handlers = append(matchedHandlers, handlers...)
	}

	for currentIndex, runeStr := range route {
		var word string = string(runeStr)

		if t.node[word] == nil {

			// If key haven't existed
			// in node map
			// create new one
			var newTrie *trie = new(trie)
			newTrie.node = make(map[string]*trie)
			t.node[word] = newTrie
		} else if t.node[slash] != nil {
			// fmt.Println(len(t.node[slash].params))
			// fmt.Println(route[currentIndex:])
			// fmt.Println(word)
			// If after slash still have word
			// then check conflict
			// err := t.checkConflictWildcard(route, currentIndex)
			// if err != nil {
			// 	panic(err)
			// }
		}

		// Pass route route after "*"
		// to help we know whether has suffix or not
		if word == wildcard {
			var remainRoute string = route[currentIndex+1:]
			var slashIndex int = strings.Index(remainRoute, slash)
			var suffixKey string

			// Treating the string from "*" to "/" is suffix
			// therefore one router can have many suffix
			if slashIndex > -1 {
				suffixKey = remainRoute[:slashIndex]
			} else {
				suffixKey = remainRoute
			}
			if suffixKey != empty {
				t.suffix = append(t.suffix, suffixKey)
			}
		}

		// Pass params and method to node has key = ":"
		// to easy to access params
		if word == param {
			if t.params == nil {
				params := make(map[string]string)
				t.params = params
			}

			// Handle case has many params
			// get param from first index to slash index
			var remainRoute string = route[currentIndex+1:]
			var slashIndex int = strings.Index(remainRoute, slash)

			if slashIndex > -1 {
				t.params[httpMethod] = remainRoute[:slashIndex]
			} else {
				t.params[httpMethod] = remainRoute
			}
		}

		// When loop runs to last index
		// flag isEnd will set true
		// add http method to map
		// add handle request function to map
		if currentIndex == lastIndex {
			t.node[word].isEnd = true
			t.node[word].handlers = handlers
		}
		t = t.node[word]
	}
}

// Check client send path if match in trie
func (t *trie) match(path, httpMethod string, params map[string]string) (bool, []Handler) {
	var lastIndex int = len(path) - 1
	var remainPath string
	var matched bool
	var handlers []Handler

	for currentIndex, runeStr := range path {
		var word string = string(runeStr)
		if t.node[word] != nil {

			// If match whole path (loop no break and isEnd = true)
			// return handler functions is matched
			if currentIndex == lastIndex && t.node[word].isEnd {
				matched = true
				handlers = t.node[word].handlers
			}
			t = t.node[word]

			// If route haven't matched
			// keep the remain path to check once more time with below logic
		} else {
			remainPath = path[currentIndex:]
			break
		}
	}

	// With remain path, divide into 2 cases:
	// #CASE 1 router includes params with matched HTTP method
	// so remain path start with ":"
	// check whether path variables existed
	if !matched && t.node[param] != nil && t.params[httpMethod] != empty {
		var paramValue string

		// A param consider from ":" to first "/"
		// after get param, remain path will
		// replaced params with ":<key_params>"
		// then run recursively with remain
		// till remain string matched or unmatched both case
		var slashIndex int = strings.Index(remainPath, slash)

		// If slash index > -1 mean path maybe have more than 1 params
		if slashIndex > -1 {
			paramValue = remainPath[0:slashIndex]
			remainPath = param + t.params[httpMethod] + remainPath[slashIndex:]
		} else {
			paramValue = remainPath[0:]
			remainPath = param + t.params[httpMethod]
		}

		// Put param value to req.Params
		params[t.params[httpMethod]] = paramValue
		matched, handlers = t.match(remainPath, httpMethod, params) // Recursive

		// #CASE 2 router includes "*"
		// check whether any string pattern existed
	} else if !matched && t.node[wildcard] != nil {

		// Suffix is an string array
		// check whether any suffix match with path client send
		var suffixIndex int = -1
		if len(t.suffix) > 0 {
			var slashIndex int = strings.Index(remainPath, slash)
			var tempPath string = remainPath
			if slashIndex > -1 {
				tempPath = remainPath[:slashIndex]
			}
			for _, suffix := range t.suffix {
				suffixIndex = strings.Index(tempPath, suffix)
				if suffixIndex > -1 {
					break
				}
			}
		}

		// 3 conditional statements below
		// solve 3 cases:
		// 1 "/before_*_after" => _after "*" is suffix
		// 2 "/before_*/after" => hasn't suffix but "*" not placed at last index
		// 3 "/before_*" => hasn't suffix but "*" placed at last index

		// After "*" has suffix
		if suffixIndex > -1 {
			remainPath = wildcard + remainPath[suffixIndex:]

			// After "*" hasn't suffix
		} else {
			var slashIndex int = strings.Index(remainPath, slash)

			// "*" not placed at last index
			if slashIndex > -1 {
				remainPath = wildcard + remainPath[slashIndex:]

				// "*" placed at last index
			} else {
				remainPath = wildcard
			}
		}

		matched, handlers = t.match(remainPath, httpMethod, params) // Recursive
	}
	return matched, handlers
}
