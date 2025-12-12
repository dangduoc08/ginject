package routing

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/utils"
)

func TestTrieLen(t *testing.T) {
	paths := []string{
		"/users/{userId}/",
		"/feeds/all/",
		"/users/{userId}/friends/all/",
	}
	tr := NewTrie()

	for _, path := range paths {
		tr.insert(path, '/', -1, nil, nil)
	}

	expected1 := 6
	actual1 := tr.len()
	if actual1 != expected1 {
		t.Error(utils.ErrorMessage(actual1, expected1, "trie length should be equal"))
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
		tr.insert(path, '/', i, nil, nil)
	}

	actual1 := tr.Children["users"]
	if actual1 == nil {
		t.Error(utils.ErrorMessage(actual1, nil, "trie node should not be null"))
	}

	actual2 := tr.Children["users"].Children["{userId}"]
	if actual2 == nil {
		t.Error(utils.ErrorMessage(actual2, nil, "trie node should not be null"))
	}

	actual3 := tr.Children["users"].Children["{userId}"].Children["friends"]
	if actual3 == nil {
		t.Error(utils.ErrorMessage(actual3, nil, "trie node should not be null"))
	}

	actual4 := tr.Children["feeds"]
	if actual4 == nil {
		t.Error(utils.ErrorMessage(actual4, nil, "trie node should not be null"))
	}

	actual5 := tr.Children["feeds"].Children["all"]
	if actual5 == nil {
		t.Error(utils.ErrorMessage(actual5, nil, "trie node should not be null"))
	}

	actual6 := tr.Children["users"].Children["{userId}"].Children["friends"].Index
	expected6 := -1
	if actual6 != expected6 {
		t.Error(utils.ErrorMessage(actual6, expected6, "trie node index should be equal"))
	}

	actual7 := tr.Children["users"].Children["{userId}"].Children["friends"].Children["all"].Index
	expected7 := 2
	if actual7 != expected7 {
		t.Error(utils.ErrorMessage(actual7, expected7, "trie node index should be equal"))
	}
}

func TestTrieFind(t *testing.T) {
	cases := []string{
		fmt.Sprintf("/users/$/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/feeds/all/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/users/$/friends/$/%v/", fromMethodtoPattern(http.MethodGet)),
		fmt.Sprintf("/*/feeds/{feed*Id}/*/files/*.html/*/%v/", fromMethodtoPattern(http.MethodGet)),
	}
	tr := NewTrie()

	for i, path := range cases {
		tr.insert(path, '/', i, nil, nil)
	}

	userId1 := "633b0aa5d7fc3578b655b9bd"
	friendId1 := "633b0af45f4fe7d45b00fba5"
	testPath1 := fmt.Sprintf("/users/%v/friends/%v/[%v]/", userId1, friendId1, http.MethodGet)

	actualIndex1, _, actualParams1, _ := tr.find(testPath1, http.MethodGet, "", '/')
	expectedIndex1 := 2
	if actualIndex1 != expectedIndex1 {
		t.Error(utils.ErrorMessage(actualIndex1, expectedIndex1, "trie node index should be equal"))
	}

	if actualParams1[0] != userId1 {
		t.Error(utils.ErrorMessage(actualParams1[0], userId1, "trie param should be equal"))
	}

	if actualParams1[1] != friendId1 {
		t.Error(utils.ErrorMessage(actualParams1[1], friendId1, "trie param should be equal"))
	}

	testPath2 := fmt.Sprintf("/users/%v/friends/[%v]/", userId1, http.MethodGet)
	actualIndex2, _, _, _ := tr.find(testPath2, http.MethodGet, "", '/')
	expectedIndex2 := -1
	if actualIndex2 != expectedIndex2 {
		t.Error(utils.ErrorMessage(actualIndex2, expectedIndex2, "trie node index should be equal"))
	}

	testPath3 := fmt.Sprintf("/api/feeds/{feedApiId}/next/files/index.html/endpoint/[%v]/", http.MethodGet)
	actualIndex3, _, _, _ := tr.find(testPath3, http.MethodGet, "", '/')
	expectedIndex3 := 3
	if actualIndex3 != expectedIndex3 {
		t.Error(utils.ErrorMessage(actualIndex3, expectedIndex3, "trie node index should be equal"))
	}

	testPath4 := fmt.Sprintf("/api/feeds/{feedApiId}/next/files/index.html/endpoint/any/things/after/[%v]/", http.MethodGet)
	actualIndex4, _, _, _ := tr.find(testPath4, http.MethodGet, "", '/')
	expectedIndex4 := 3
	if actualIndex4 != expectedIndex4 {
		t.Error(utils.ErrorMessage(actualIndex4, expectedIndex4, "trie node index should be equal"))
	}
}
