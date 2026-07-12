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
		tr.Insert(path, path, '/')
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

	for _, path := range routes {
		tr.Insert(path, path, '/')
	}

	cases := []struct {
		name       string
		node       *Trie
		wantIsEnd  bool
		checkIsEnd bool
	}{
		{name: "users", node: tr.Children["users"]},
		{name: "users/{userId}", node: tr.Children["users"].Children["{userId}"]},
		{name: "users/{userId}/friends", node: tr.Children["users"].Children["{userId}"].Children["friends"], wantIsEnd: false, checkIsEnd: true},
		{name: "feeds", node: tr.Children["feeds"]},
		{name: "feeds/all", node: tr.Children["feeds"].Children["all"]},
		{name: "users/{userId}/friends/all", node: tr.Children["users"].Children["{userId}"].Children["friends"].Children["all"], wantIsEnd: true, checkIsEnd: true},
	}

	for _, c := range cases {
		if c.node == nil {
			t.Error(test.DiffMessage(c.node, nil, c.name+": trie node should not be null"))
			continue
		}
		if c.checkIsEnd && c.node.IsEnd != c.wantIsEnd {
			t.Error(test.DiffMessage(c.node.IsEnd, c.wantIsEnd, c.name+": trie node IsEnd should be equal"))
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

	for _, path := range routes {
		tr.Insert(path, path, '/')
	}

	userID1 := "633b0aa5d7fc3578b655b9bd"
	friendID1 := "633b0af45f4fe7d45b00fba5"

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
			testPath:   "/users/" + userID1 + "/friends/" + friendID1 + "/",
			wantRaw:    routes[2],
			wantParams: []string{userID1, friendID1},
		},
		{
			name:        "incomplete path should not match",
			testPath:    "/users/" + userID1 + "/friends/",
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
		actualRaw, actualWildcard, actualParams := tr.Find(c.testPath, '/', true)

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

func TestTrieFind_ParamSupportDisabled(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/users/$/", "/users/$/", '/')

	matchedRaw, _, params := tr.Find("/users/123/", '/', false)
	if matchedRaw != "" || params != nil {
		t.Error(test.DiffMessage(matchedRaw, "", "Find(path, sep, false) must not treat $ as a param placeholder"))
	}

	matchedRaw, _, _ = tr.Find("/users/$/", '/', false)
	if matchedRaw != "/users/$/" {
		t.Error(test.DiffMessage(matchedRaw, "/users/$/", "$ should still match literally when param support is disabled"))
	}
}

func TestTrieFind_ParamSupportEnabled(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/users/$/", "/users/$/", '/')

	matchedRaw, _, params := tr.Find("/users/123/", '/', true)
	if matchedRaw != "/users/$/" || len(params) != 1 || params[0] != "123" {
		t.Error(test.DiffMessage(matchedRaw, "/users/$/", "Find(path, sep, true) must capture $ as a param placeholder"))
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
	for _, path := range tc.routes {
		tr.Insert(path, path, '/')
	}

	actualRaw, actualWildcard, _ := tr.Find(tc.testPath, '/', false)
	if actualRaw != tc.wantRaw && actualWildcard != tc.wantRaw {
		t.Error(test.DiffMessage(actualRaw, tc.wantRaw, "should fall back to /lv1/* despite unrelated deeper sibling"))
	}
}

func TestTrieRemove_ExactMatch(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/users/$/", "/users/$/", '/')
	tr.Insert("/feeds/all/", "/feeds/all/", '/')

	ok := tr.Remove("/feeds/all/", '/')
	if !ok {
		t.Error(test.DiffMessage(ok, true, "Remove should report success for a previously inserted path"))
	}

	matchedRaw, _, _ := tr.Find("/feeds/all/", '/', false)
	if matchedRaw != "" {
		t.Error(test.DiffMessage(matchedRaw, "", "removed path should no longer be found"))
	}

	matchedRaw, _, _ = tr.Find("/users/123/", '/', true)
	if matchedRaw != "/users/$/" {
		t.Error(test.DiffMessage(matchedRaw, "/users/$/", "unrelated path should still match after removal"))
	}
}

func TestTrieRemove_PrunesDeadBranch(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/a/b/c/", "/a/b/c/", '/')

	ok := tr.Remove("/a/b/c/", '/')
	if !ok {
		t.Error(test.DiffMessage(ok, true, "Remove should report success"))
	}

	if got := tr.Len(); got != 0 {
		t.Error(test.DiffMessage(got, 0, "removing the only path should prune every dead ancestor node"))
	}
}

func TestTrieRemove_KeepsSharedPrefix(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/a/b/", "/a/b/", '/')
	tr.Insert("/a/c/", "/a/c/", '/')

	ok := tr.Remove("/a/b/", '/')
	if !ok {
		t.Error(test.DiffMessage(ok, true, "Remove should report success"))
	}

	matchedRaw, _, _ := tr.Find("/a/c/", '/', false)
	if matchedRaw != "/a/c/" {
		t.Error(test.DiffMessage(matchedRaw, "/a/c/", "sibling path sharing a prefix must survive removal"))
	}

	wantLen := 2 // "a" and "c" remain; "b" is pruned
	if got := tr.Len(); got != wantLen {
		t.Error(test.DiffMessage(got, wantLen, "shared prefix node must not be pruned while a sibling still uses it"))
	}
}

func TestTrieRemove_NoMatch_ReturnsFalse(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/a/b/", "/a/b/", '/')

	cases := []string{
		"/x/y/",   // segment never inserted
		"/a/",     // intermediate node, never a terminal node
		"",        // empty input
		"noslash", // no separator at all
	}

	for _, c := range cases {
		if ok := tr.Remove(c, '/'); ok {
			t.Error(test.DiffMessage(ok, false, "Remove("+c+") should report false"))
		}
	}

	matchedRaw, _, _ := tr.Find("/a/b/", '/', false)
	if matchedRaw != "/a/b/" {
		t.Error(test.DiffMessage(matchedRaw, "/a/b/", "failed Remove calls must not mutate the trie"))
	}
}

func TestTrieRemove_AlreadyRemoved_ReturnsFalse(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/a/b/", "/a/b/", '/')

	if ok := tr.Remove("/a/b/", '/'); !ok {
		t.Error(test.DiffMessage(ok, true, "first Remove should succeed"))
	}
	if ok := tr.Remove("/a/b/", '/'); ok {
		t.Error(test.DiffMessage(ok, false, "second Remove of the same path should report false"))
	}
}

type trieJSONNode struct {
	Path     string         `json:"path"`
	IsEnd    bool           `json:"isEnd"`
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
	tr.Insert("/users/$/", "/users/$/", '/')
	tr.Insert("/feeds/all/", "/feeds/all/", '/')

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
		if usersNode.IsEnd {
			t.Error(test.DiffMessage(usersNode.IsEnd, false, "users node IsEnd should be equal"))
		}

		paramNode := findChildByPath(usersNode.Children, "$")
		if paramNode == nil {
			t.Error(test.DiffMessage(paramNode, "$", "$ node should exist"))
		} else if !paramNode.IsEnd {
			t.Error(test.DiffMessage(paramNode.IsEnd, true, "$ node IsEnd should be equal"))
		}
	}

	feedsNode := findChildByPath(root.Children, "feeds")
	if feedsNode == nil {
		t.Error(test.DiffMessage(feedsNode, "feeds", "feeds node should exist"))
	} else {
		allNode := findChildByPath(feedsNode.Children, "all")
		if allNode == nil {
			t.Error(test.DiffMessage(allNode, "all", "all node should exist"))
		} else if !allNode.IsEnd {
			t.Error(test.DiffMessage(allNode.IsEnd, true, "all node IsEnd should be equal"))
		}
	}
}
