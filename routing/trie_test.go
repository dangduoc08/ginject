package routing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestTrieLenEmpty(t *testing.T) {
	tr := NewTrie()

	actual := tr.len()
	expected := 0
	if actual != expected {
		t.Error(test.DiffMessage(actual, expected, "empty trie length should be equal"))
	}
}

func TestTrieLen(t *testing.T) {
	paths := []string{
		"/users/{userId}/",
		"/feeds/all/",
		"/users/{userId}/friends/all/",
	}
	tr := NewTrie()

	for _, path := range paths {
		tr.insert(path, path, '/', -1)
	}

	expected1 := 6
	actual1 := tr.len()
	if actual1 != expected1 {
		t.Error(test.DiffMessage(actual1, expected1, "trie length should be equal"))
	}
}

func TestTrieInsert(t *testing.T) {
	cases := []string{
		"/users/{userId}/",
		"/feeds/all/",
		"/users/{userId}/friends/all/",
	}
	tr := NewTrie()

	for i, path := range cases {
		tr.insert(path, path, '/', i)
	}

	actual1 := tr.Children["users"]
	if actual1 == nil {
		t.Error(test.DiffMessage(actual1, nil, "trie node should not be null"))
	}

	actual2 := tr.Children["users"].Children["{userId}"]
	if actual2 == nil {
		t.Error(test.DiffMessage(actual2, nil, "trie node should not be null"))
	}

	actual3 := tr.Children["users"].Children["{userId}"].Children["friends"]
	if actual3 == nil {
		t.Error(test.DiffMessage(actual3, nil, "trie node should not be null"))
	}

	actual4 := tr.Children["feeds"]
	if actual4 == nil {
		t.Error(test.DiffMessage(actual4, nil, "trie node should not be null"))
	}

	actual5 := tr.Children["feeds"].Children["all"]
	if actual5 == nil {
		t.Error(test.DiffMessage(actual5, nil, "trie node should not be null"))
	}

	actual6 := tr.Children["users"].Children["{userId}"].Children["friends"].Index
	expected6 := -1
	if actual6 != expected6 {
		t.Error(test.DiffMessage(actual6, expected6, "trie node index should be equal"))
	}

	actual7 := tr.Children["users"].Children["{userId}"].Children["friends"].Children["all"].Index
	expected7 := 2
	if actual7 != expected7 {
		t.Error(test.DiffMessage(actual7, expected7, "trie node index should be equal"))
	}
}

func TestTrieFind(t *testing.T) {
	cases := []string{
		fmt.Sprintf("/users/$/%v/", toPattern(http.MethodGet, "[", "]")),
		fmt.Sprintf("/feeds/all/%v/", toPattern(http.MethodGet, "[", "]")),
		fmt.Sprintf("/users/$/friends/$/%v/", toPattern(http.MethodGet, "[", "]")),
		fmt.Sprintf("/*/feeds/{feed*Id}/*/files/*.html/*/%v/", toPattern(http.MethodGet, "[", "]")),
	}
	tr := NewTrie()

	for i, path := range cases {
		tr.insert(path, path, '/', i)
	}

	userId1 := "633b0aa5d7fc3578b655b9bd"
	friendId1 := "633b0af45f4fe7d45b00fba5"
	testPath1 := fmt.Sprintf("/users/%v/friends/%v/[%v]/", userId1, friendId1, http.MethodGet)

	actualIndex1, _, actualParams1 := tr.find(testPath1, http.MethodGet, "", '/')

	expectedIndex1 := 2
	if actualIndex1 != expectedIndex1 {
		t.Error(test.DiffMessage(actualIndex1, expectedIndex1, "trie node index should be equal"))
	}

	if actualParams1[0] != userId1 {
		t.Error(test.DiffMessage(actualParams1[0], userId1, "trie param should be equal"))
	}

	if actualParams1[1] != friendId1 {
		t.Error(test.DiffMessage(actualParams1[1], friendId1, "trie param should be equal"))
	}

	testPath2 := fmt.Sprintf("/users/%v/friends/[%v]/", userId1, http.MethodGet)
	actualIndex2, _, _ := tr.find(testPath2, http.MethodGet, "", '/')
	expectedIndex2 := -1
	if actualIndex2 != expectedIndex2 {
		t.Error(test.DiffMessage(actualIndex2, expectedIndex2, "trie node index should be equal"))
	}

	testPath3 := fmt.Sprintf("/api/feeds/{feedApiId}/next/files/index.html/endpoint/[%v]/", http.MethodGet)
	actualIndex3, _, _ := tr.find(testPath3, http.MethodGet, "", '/')
	expectedIndex3 := 3
	if actualIndex3 != expectedIndex3 {
		t.Error(test.DiffMessage(actualIndex3, expectedIndex3, "trie node index should be equal"))
	}

	testPath4 := fmt.Sprintf("/api/feeds/{feedApiId}/next/files/index.html/endpoint/any/things/after/[%v]/", http.MethodGet)
	actualIndex4, _, _ := tr.find(testPath4, http.MethodGet, "", '/')
	expectedIndex4 := 3
	if actualIndex4 != expectedIndex4 {
		t.Error(test.DiffMessage(actualIndex4, expectedIndex4, "trie node index should be equal"))
	}
}

func TestTrieFindWildcardFallbackThroughUnrelatedSibling(t *testing.T) {
	cases := []string{
		fmt.Sprintf("/lv1/*/%v/", toPattern(http.MethodGet, "[", "]")),
		fmt.Sprintf("/lv1/lv2/lv3/pong/%v/", toPattern(http.MethodGet, "[", "]")),
	}
	tr := NewTrie()

	for i, path := range cases {
		tr.insert(path, path, '/', i)
	}

	testPath := fmt.Sprintf("/lv1/lv2/lv3/extra/[%v]/", http.MethodGet)
	actualIndex, _, _ := tr.find(testPath, http.MethodGet, "", '/')
	expectedIndex := 0
	if actualIndex != expectedIndex {
		t.Error(test.DiffMessage(actualIndex, expectedIndex, "should fall back to /lv1/* despite unrelated deeper sibling"))
	}
}

func findChildByPath(children []any, path string) map[string]any {
	for _, c := range children {
		child := c.(map[string]any)
		if child["path"] == path {
			return child
		}
	}
	return nil
}

func TestTrieToJSON(t *testing.T) {
	tr := NewTrie()
	tr.insert("/users/$/", "/users/$/", '/', 0)
	tr.insert("/feeds/all/", "/feeds/all/", '/', 1)

	js, err := tr.ToJSON()
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "ToJSON should not error"))
	}

	var root map[string]any
	if err := json.Unmarshal([]byte(js), &root); err != nil {
		t.Error(test.DiffMessage(err, nil, "ToJSON output should be valid JSON"))
	}

	rootChildren := root["children"].([]any)
	expectedRootChildren := 2
	if len(rootChildren) != expectedRootChildren {
		t.Error(test.DiffMessage(len(rootChildren), expectedRootChildren, "root children count should be equal"))
	}

	usersNode := findChildByPath(rootChildren, "users")
	if usersNode == nil {
		t.Error(test.DiffMessage(usersNode, "users", "users node should exist"))
	} else {
		if usersNode["index"].(float64) != -1 {
			t.Error(test.DiffMessage(usersNode["index"], -1, "users node index should be equal"))
		}

		paramNode := findChildByPath(usersNode["children"].([]any), "$")
		if paramNode == nil {
			t.Error(test.DiffMessage(paramNode, "$", "$ node should exist"))
		} else if paramNode["index"].(float64) != 0 {
			t.Error(test.DiffMessage(paramNode["index"], 0, "$ node index should be equal"))
		}
	}

	feedsNode := findChildByPath(rootChildren, "feeds")
	if feedsNode == nil {
		t.Error(test.DiffMessage(feedsNode, "feeds", "feeds node should exist"))
	} else {
		allNode := findChildByPath(feedsNode["children"].([]any), "all")
		if allNode == nil {
			t.Error(test.DiffMessage(allNode, "all", "all node should exist"))
		} else if allNode["index"].(float64) != 1 {
			t.Error(test.DiffMessage(allNode["index"], 1, "all node index should be equal"))
		}
	}
}
