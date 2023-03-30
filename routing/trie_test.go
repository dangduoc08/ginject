package routing

import (
	"fmt"
	"testing"
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

	expect1 := 6
	output1 := tr.len()
	if output1 != expect1 {
		t.Errorf("tr.len() = %v; expect = %v", output1, expect1)
	}
}

func TestTrieInsert(t *testing.T) {
	paths := []string{
		"/users/{userId}/",
		"/feeds/all/",
		"/users/{userId}/friends/all/",
	}
	tr := NewTrie()

	for i, path := range paths {
		tr.insert(path, '/', i, nil, nil)
	}

	output1 := tr.Children["users"]
	if output1 == nil {
		t.Errorf("tr.Children[\"users\"] = %v; expect ≠ %v", output1, nil)
	}

	output2 := tr.Children["users"].Children["{userId}"]
	if output2 == nil {
		t.Errorf("tr.Children[\"users\"].Children[\"{userId}\"] = %v; expect ≠ %v", output2, nil)
	}

	output3 := tr.Children["users"].Children["{userId}"].Children["friends"]
	if output3 == nil {
		t.Errorf("tr.Children[\"users\"].Children[\"{userId}\"].Children[\"friends\"] = %v; expect ≠ %v", output3, nil)
	}

	output4 := tr.Children["feeds"]
	if output4 == nil {
		t.Errorf("tr.Children[\"feeds\"] = %v; expect ≠ %v", output4, nil)
	}

	output5 := tr.Children["feeds"].Children["all"]
	if output5 == nil {
		t.Errorf("tr.Children[\"feeds\"].Children[\"all\"] = %v; expect ≠ %v", output5, nil)
	}

	output6 := tr.Children["users"].Children["{userId}"].Children["friends"].Index
	expect1 := -1
	if output6 != expect1 {
		t.Errorf("tr.Children[\"users\"].Children[\"{userId}\"].Children[\"friends\"].Index = %v; expect = %v", output6, expect1)
	}

	output7 := tr.Children["users"].Children["{userId}"].Children["friends"].Children["all"].Index
	expect7 := 2
	if output7 != expect7 {
		t.Errorf("tr.Children[\"users\"].Children[\"{userId}\"].Children[\"friends\"].Children[\"all\"].Index = %v; expect = %v", output7, expect7)
	}
}

func TestTrieFind(t *testing.T) {
	paths := []string{
		"/users/$/",
		"/feeds/all/",
		"/users/$/friends/$/",
		"/*/feeds/{feed*Id}/*/files/*.html/*/",
	}
	tr := NewTrie()

	for i, path := range paths {
		tr.insert(path, '/', i, nil, nil)
	}

	userId1 := "633b0aa5d7fc3578b655b9bd"
	friendId1 := "633b0af45f4fe7d45b00fba5"
	testPath1 := fmt.Sprintf("/users/%v/friends/%v/", userId1, friendId1)

	index1, _, params1, _ := tr.find(testPath1, '/')
	expectIndex1 := 2
	if index1 != expectIndex1 {
		t.Errorf("tr.find(%v), '/') return Index = %v; expect = %v", testPath1, index1, expectIndex1)
	}

	if params1[0] != userId1 {
		t.Errorf("params1[0] = %v; expect = %v", params1[0], userId1)
	}

	if params1[1] != friendId1 {
		t.Errorf("params1[1] = %v; expect = %v", params1[1], friendId1)
	}

	testPath2 := fmt.Sprintf("/users/%v/friends", userId1)
	index2, _, _, _ := tr.find(testPath2, '/')
	expectIndex2 := -1
	if index2 != expectIndex2 {
		t.Errorf("tr.find(%v), '/') return Index = %v; expect = %v", testPath2, index2, expectIndex2)
	}

	index3, _, _, _ := tr.find("/api/feeds/{feedApiId}/next/files/index.html/endpoint/", '/')
	expectIndex3 := 3
	if index3 != expectIndex3 {
		t.Errorf("tr.find(\"/api/feeds/{feedApiId}/next/files/index.html/endpoint/\", '/') return Index = %v; expect = %v", index3, expectIndex3)
	}

	index4, _, _, _ := tr.find("/api/feeds/{feedApiId}/next/files/index.html/endpoint/any/things/after/", '/')
	expectIndex4 := 3
	if index4 != expectIndex4 {
		t.Errorf("tr.find(\"/api/feeds/{feedApiId}/next/files/index.html/endpoint/any/things/after/\", '/') return Index = %v; expect = %v", index3, expectIndex3)
	}
}

func TestTrieScan(t *testing.T) {
	paths := []string{
		"/users/$/",
		"/feeds/all/",
		"/feeds/",
		"/users/$/friends/$/",
		"/*/feeds/{feed*Id}/*/files/*.html/*/",
	}
	tr := NewTrie()

	for i, path := range paths {
		tr.insert(path, '/', i, nil, nil)
	}

	tr.scan(func(node *Trie) {

	})
}
