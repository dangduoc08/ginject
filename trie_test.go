package gogo

import (
	"net/http"
	"testing"
)

var t trie = trie{
	node: make(map[string]*trie),
}

var handler Handler = func(req *Request, res ResponseExtender, next func()) {}

func init() {
	t.insert("/path_1", http.MethodGet, handler)
	t.insert("/path_1/:param_1/path_2/:param_2", http.MethodGet, handler)
	t.insert("*/:param_1/*/path_1/:param_2", http.MethodGet, handler)
	t.insert("*/*/*", http.MethodGet, handler)
}

func TestAbsoluteURL(test *testing.T) {
	params := make(map[string]string)
	matched, _ := t.match("/path_1", http.MethodGet, &params)

	if matched != true {
		test.Errorf("Test absolute URL couldn't matched")
	}
}

func TestURLWithParams(test *testing.T) {
	params := make(map[string]string)
	matched, _ := t.match("/path_1/foo/path_2/bar", http.MethodGet, &params)
	var err string = "Test URL with params "

	if matched != true {
		err = err + "couldn't matched"
		test.Errorf(err)
	}

	if params["param_1"] != "foo" || params["param_2"] != "bar" {
		err = err + "couldn't get params"
		test.Errorf(err)
	}
}

func TestURLWithAnyPatternAndParams(test *testing.T) {
	params := make(map[string]string)
	matched, _ := t.match("/any_1/foo/any_2/path_1/bar", http.MethodGet, &params)
	var err string = "Test URL with any pattern and params "

	if matched != true {
		err = err + "couldn't matched"
		test.Errorf(err)
	}

	if params["param_1"] != "foo" || params["param_2"] != "bar" {
		err = err + "couldn't get params"
		test.Errorf(err)
	}
}

func TestURLWithAnyAnyAny(test *testing.T) {
	params := make(map[string]string)
	matched, _ := t.match("/any_1/any_2/any_3", http.MethodGet, &params)
	var err string = "Test URL with any any any pattern "

	if matched != true {
		err = err + "couldn't matched"
		test.Errorf(err)
	}
}
