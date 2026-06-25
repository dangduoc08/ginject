package ds

import (
	"encoding/json"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestTrieLenEmpty(t *testing.T) {
	tr := NewTrie()

	actual := tr.Len()
	expected := 0
	if actual != expected {
		t.Error(test.DiffMessage(actual, expected, "empty trie length should be equal"))
	}
}

func TestTrieLen(t *testing.T) {
	tc := struct {
		paths    []string
		expected int
	}{
		paths: []string{
			"/users/{userId}/",
			"/feeds/all/",
			"/users/{userId}/friends/all/",
		},
		expected: 6,
	}

	tr := NewTrie()
	for _, path := range tc.paths {
		tr.Insert(path, path, '/', -1)
	}

	actual := tr.Len()
	if actual != tc.expected {
		t.Error(test.DiffMessage(actual, tc.expected, "trie length should be equal"))
	}
}

func TestTrieInsert(t *testing.T) {
	routes := []string{
		"/users/{userId}/",
		"/feeds/all/",
		"/users/{userId}/friends/all/",
	}
	tr := NewTrie()

	for i, path := range routes {
		tr.Insert(path, path, '/', i)
	}

	wantID2 := -1
	wantID7 := 2
	cases := []struct {
		name   string
		node   *Trie
		wantID *int
	}{
		{name: "users", node: tr.Children["users"]},
		{name: "users/{userId}", node: tr.Children["users"].Children["{userId}"]},
		{name: "users/{userId}/friends", node: tr.Children["users"].Children["{userId}"].Children["friends"], wantID: &wantID2},
		{name: "feeds", node: tr.Children["feeds"]},
		{name: "feeds/all", node: tr.Children["feeds"].Children["all"]},
		{name: "users/{userId}/friends/all", node: tr.Children["users"].Children["{userId}"].Children["friends"].Children["all"], wantID: &wantID7},
	}

	for _, c := range cases {
		if c.node == nil {
			t.Error(test.DiffMessage(c.node, nil, c.name+": trie node should not be null"))
			continue
		}
		if c.wantID != nil && c.node.ID != *c.wantID {
			t.Error(test.DiffMessage(c.node.ID, *c.wantID, c.name+": trie node id should be equal"))
		}
	}
}

func TestTrieFind(t *testing.T) {
	routes := []string{
		"/users/$/",
		"/feeds/all/",
		"/users/$/friends/$/",
		"/*/feeds/{feed*Id}/*/files/*.html/*/",
	}
	tr := NewTrie()

	for i, path := range routes {
		tr.Insert(path, path, '/', i)
	}

	userId1 := "633b0aa5d7fc3578b655b9bd"
	friendId1 := "633b0af45f4fe7d45b00fba5"

	cases := []struct {
		name              string
		testPath          string
		wantRaw           string
		acceptWildcardRaw bool
		wantNoMatch       bool
		wantParams        []string
	}{
		{
			name:       "deep param match",
			testPath:   "/users/" + userId1 + "/friends/" + friendId1 + "/",
			wantRaw:    routes[2],
			wantParams: []string{userId1, friendId1},
		},
		{
			name:        "incomplete path should not match",
			testPath:    "/users/" + userId1 + "/friends/",
			wantNoMatch: true,
		},
		{
			name:              "wildcard deep match, exact length",
			testPath:          "/api/feeds/{feedApiId}/next/files/id.html/endpoint/",
			wantRaw:           routes[3],
			acceptWildcardRaw: true,
		},
		{
			name:              "wildcard deep match, extra trailing segments",
			testPath:          "/api/feeds/{feedApiId}/next/files/id.html/endpoint/any/things/after/",
			wantRaw:           routes[3],
			acceptWildcardRaw: true,
		},
	}

	for _, c := range cases {
		_, actualRaw, _, actualWildcard, actualParams := tr.Find(c.testPath, '/')

		if c.wantNoMatch {
			if actualRaw != "" || actualWildcard != "" {
				t.Error(test.DiffMessage(actualRaw, "", c.name))
			}
			continue
		}

		matched := actualRaw == c.wantRaw || (c.acceptWildcardRaw && actualWildcard == c.wantRaw)
		if !matched {
			t.Error(test.DiffMessage(actualRaw, c.wantRaw, c.name))
		}

		for i, want := range c.wantParams {
			if actualParams[i] != want {
				t.Error(test.DiffMessage(actualParams[i], want, c.name+": param should be equal"))
			}
		}
	}
}

func TestTrieFindWildcardFallbackThroughUnrelatedSibling(t *testing.T) {
	tc := struct {
		routes   []string
		testPath string
		wantRaw  string
	}{
		routes: []string{
			"/lv1/*/",
			"/lv1/lv2/lv3/pong/",
		},
		testPath: "/lv1/lv2/lv3/extra/",
		wantRaw:  "/lv1/*/",
	}

	tr := NewTrie()
	for i, path := range tc.routes {
		tr.Insert(path, path, '/', i)
	}

	_, actualRaw, _, actualWildcard, _ := tr.Find(tc.testPath, '/')
	if actualRaw != tc.wantRaw && actualWildcard != tc.wantRaw {
		t.Error(test.DiffMessage(actualRaw, tc.wantRaw, "should fall back to /lv1/* despite unrelated deeper sibling"))
	}
}

type trieJSONNode struct {
	Path     string         `json:"path"`
	ID       int            `json:"id"`
	Children []trieJSONNode `json:"children"`
}

func findChildByPath(children []trieJSONNode, path string) *trieJSONNode {
	for i := range children {
		if children[i].Path == path {
			return &children[i]
		}
	}
	return nil
}

func TestTrieToJSON(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/users/$/", "/users/$/", '/', 0)
	tr.Insert("/feeds/all/", "/feeds/all/", '/', 1)

	js, err := tr.ToJSON()
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "ToJSON should not error"))
	}

	var root trieJSONNode
	if err := json.Unmarshal([]byte(js), &root); err != nil {
		t.Error(test.DiffMessage(err, nil, "ToJSON output should be valid JSON"))
	}

	expectedRootChildren := 2
	if len(root.Children) != expectedRootChildren {
		t.Error(test.DiffMessage(len(root.Children), expectedRootChildren, "root children count should be equal"))
	}

	usersNode := findChildByPath(root.Children, "users")
	if usersNode == nil {
		t.Error(test.DiffMessage(usersNode, "users", "users node should exist"))
	} else {
		if usersNode.ID != -1 {
			t.Error(test.DiffMessage(usersNode.ID, -1, "users node id should be equal"))
		}

		paramNode := findChildByPath(usersNode.Children, "$")
		if paramNode == nil {
			t.Error(test.DiffMessage(paramNode, "$", "$ node should exist"))
		} else if paramNode.ID != 0 {
			t.Error(test.DiffMessage(paramNode.ID, 0, "$ node id should be equal"))
		}
	}

	feedsNode := findChildByPath(root.Children, "feeds")
	if feedsNode == nil {
		t.Error(test.DiffMessage(feedsNode, "feeds", "feeds node should exist"))
	} else {
		allNode := findChildByPath(feedsNode.Children, "all")
		if allNode == nil {
			t.Error(test.DiffMessage(allNode, "all", "all node should exist"))
		} else if allNode.ID != 1 {
			t.Error(test.DiffMessage(allNode.ID, 1, "all node id should be equal"))
		}
	}
}
