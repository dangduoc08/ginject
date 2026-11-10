package ds

import (
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/internal/test"
)

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

func TestTrieInsert_CachesParamAndWildcardChildren(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/a/$/", "/a/$/", '/')
	tr.Insert("/a/*/", "/a/*/", '/')
	a := tr.Children["a"]
	if a.param != a.Children["$"] || a.param == nil {
		t.Error(test.DiffMessage(a.param, a.Children["$"], "param pointer"))
	}
	if a.wildcard != a.Children["*"] || a.wildcard == nil {
		t.Error(test.DiffMessage(a.wildcard, a.Children["*"], "wildcard pointer"))
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
		actualRaw, actualWildcard, actualParams := tr.Find(c.testPath, '/')

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

func TestTrieFind_CapturesParam(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/users/$/", "/users/$/", '/')

	matchedRaw, _, params := tr.Find("/users/123/", '/')
	if matchedRaw != "/users/$/" || len(params) != 1 || params[0] != "123" {
		t.Error(test.DiffMessage(matchedRaw, "/users/$/", "Find must capture $ as a param placeholder"))
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

	actualRaw, actualWildcard, _ := tr.Find(tc.testPath, '/')
	if actualRaw != tc.wantRaw && actualWildcard != tc.wantRaw {
		t.Error(test.DiffMessage(actualRaw, tc.wantRaw, "should fall back to /lv1/* despite unrelated deeper sibling"))
	}
}

func TestTrieFind_PathWithoutTrailingSeparator(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/a/", "/a/", '/')
	if raw, wildcard, params := tr.Find("/a", '/'); raw != "" || wildcard != "" || params != nil {
		t.Error(test.DiffMessage([]any{raw, wildcard, params}, []any{"", "", nil}, "path must end with the separator"))
	}
}

func TestTrieFind_BacktracksFromDeadEndExactMatch(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/users/$/profile/", "/users/$/profile/", '/')
	tr.Insert("/users/me/", "/users/me/", '/')

	raw, _, params := tr.Find("/users/me/profile/", '/')
	if raw != "/users/$/profile/" || strings.Join(params, ",") != "me" {
		t.Error(test.DiffMessage([]any{raw, params}, []any{"/users/$/profile/", []string{"me"}}, "dead-end exact segment falls back to the param route"))
	}
	if raw, _, _ := tr.Find("/users/me/", '/'); raw != "/users/me/" {
		t.Error(test.DiffMessage(raw, "/users/me/", "exact route still wins"))
	}
}

func TestTrieFind_LiteralPlaceholderSegmentIsAParam(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/users/$/", "/users/$/", '/')
	tr.Insert("/files/*/", "/files/*/", '/')

	raw, _, params := tr.Find("/users/$/", '/')
	if raw != "/users/$/" || strings.Join(params, ",") != "$" {
		t.Error(test.DiffMessage([]any{raw, params}, []any{"/users/$/", []string{"$"}}, "a literal $ segment is captured as a param value"))
	}
	raw, wildcard, _ := tr.Find("/files/*/", '/')
	if raw != "/files/*/" && wildcard != "/files/*/" {
		t.Error(test.DiffMessage([]any{raw, wildcard}, "/files/*/", "a literal * segment still reaches the wildcard route"))
	}
}

func TestTrieFind_HostilePlaceholderPathIsLinear(t *testing.T) {
	const depth = 24
	tr := NewTrie()
	paramRoute := "/" + strings.Repeat("$/", depth) + "end/"
	wildcardRoute := "/" + strings.Repeat("*/", depth) + "end/"
	tr.Insert(paramRoute, paramRoute, '/')
	tr.Insert(wildcardRoute, wildcardRoute, '/')
	hostile := "/" + strings.Repeat("$/", depth) + "nope/"

	start := time.Now()
	raw, _, _ := tr.Find(hostile, '/')
	elapsed := time.Since(start)

	if raw != "" {
		t.Error(test.DiffMessage(raw, "", "hostile path must not match"))
	}
	if elapsed > 10*time.Millisecond {
		t.Error(test.DiffMessage(elapsed.String(), "< 10ms", "Find must stay linear on placeholder-shaped segments"))
	}
}

func TestTrieFind_ConcurrentReaders(t *testing.T) {
	tr := NewTrie()
	routes := []string{"/users/$/", "/users/$/friends/$/", "/feeds/all/", "/files/*.html/", "/static/*/"}
	for _, r := range routes {
		tr.Insert(r, r, '/')
	}

	cases := []struct {
		path        string
		expectedRaw string
	}{
		{"/users/7/", "/users/$/"},
		{"/users/7/friends/9/", "/users/$/friends/$/"},
		{"/feeds/all/", "/feeds/all/"},
		{"/files/index.html/", "/files/*.html/"},
	}

	const goroutines = 64
	const iterations = 500
	var wg sync.WaitGroup
	errs := make(chan string, goroutines)

	for range goroutines {
		wg.Go(func() {
			for i := range iterations {
				c := cases[i%len(cases)]
				matchedRaw, _, _ := tr.Find(c.path, '/')
				if matchedRaw != c.expectedRaw {
					errs <- test.DiffMessage(matchedRaw, c.expectedRaw, "concurrent Find("+c.path+")")
					return
				}
			}
		})
	}

	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
}

func TestMatchWildcard(t *testing.T) {
	cases := []struct {
		segment string
		route   string
		want    bool
	}{
		{"index.html", "*.html", true},
		{"image.png", "image.*", true},
		{"index.html", "in*.html", true},
		{"in.html", "in*.html", true},
		{"index.html", "*.png", false},
		{"foo", "foo", true},
		{"foobar", "foo", false},
		{"a.html.html", "*.html", true},
		{"aaaa", "*a", true},
		{"a", "*", true},
		{"", "*", true},
		{"abc", "**", true},
		{"x.y.z", "*.z", true},
		{"x.y.z", "x.*", true},
		{"abXcdYef", "ab*cd*ef", true},
		{"abcdef", "a*c*f", true},
		{"abcd", "a*x*d", false},
		{"abef", "ab*cd*ef", false},
		{"a-b-", "a*b*", true},
	}

	for _, c := range cases {
		got := matchWildcard(c.segment, c.route)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "matchWildcard("+c.segment+", "+c.route+")"))
		}
	}
}

func TestTrieFind_BacktrackKeepsParamsWithTheirRoute(t *testing.T) {
	tr := NewTrie()
	tr.Insert("/a/b/$/*/", "/a/b/$/*/", '/')
	tr.Insert("/a/$/c/d/e/", "/a/$/c/d/e/", '/')

	raw, wildcard, params := tr.Find("/a/b/c/d/e/", '/')
	masterRaw, masterWildcard, masterParams := findMaster(tr, "/a/b/c/d/e/", '/', true)
	got := []any{raw, wildcard, params}
	want := []any{masterRaw, masterWildcard, masterParams}
	if raw != masterRaw || wildcard != masterWildcard || strings.Join(params, ",") != strings.Join(masterParams, ",") {
		t.Error(test.DiffMessage(got, want, "a wildcard fallback keeps master's params"))
	}
}

func TestTrieFind_MatchesMasterWhenMasterFindsAnything(t *testing.T) {
	r := rand.New(rand.NewPCG(7, 9))
	for range 50000 {
		routes := randomRoutes(r)
		tr := NewTrie()
		for _, route := range routes {
			tr.Insert(route, route, '/')
		}
		for range 10 {
			path := randomPath(r)
			wantRaw, wantWildcard, wantParams := findMaster(tr, path, '/', true)
			if wantRaw == "" && wantWildcard == "" {
				continue
			}
			gotRaw, gotWildcard, gotParams := tr.Find(path, '/')
			got := []any{gotRaw, gotWildcard, strings.Join(gotParams, ",")}
			want := []any{wantRaw, wantWildcard, strings.Join(wantParams, ",")}
			if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
				t.Fatal(test.DiffMessage(got, want, "routes "+strings.Join(routes, " ")+" path "+path))
			}
		}
	}
}

func TestTrieFind_BacktrackingOnlyTurnsMasterMissesIntoMatches(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 5))
	recovered := 0
	for range 50000 {
		routes := randomRoutes(r)
		tr := NewTrie()
		for _, route := range routes {
			tr.Insert(route, route, '/')
		}
		for range 10 {
			path := randomPath(r)
			if masterRaw, masterWildcard, _ := findMaster(tr, path, '/', true); masterRaw != "" || masterWildcard != "" {
				continue
			}
			want, wantParams, ambiguous := bruteForceMatch(routes, path)
			if ambiguous {
				continue
			}
			got, gotWildcard, gotParams := tr.Find(path, '/')
			if got != want || gotWildcard != "" || (got != "" && strings.Join(gotParams, ",") != strings.Join(wantParams, ",")) {
				t.Fatal(test.DiffMessage([]any{got, gotWildcard, gotParams}, []any{want, "", wantParams}, "routes "+strings.Join(routes, " ")+" path "+path))
			}
			if got != "" {
				recovered++
			}
		}
	}
	if recovered == 0 {
		t.Error(test.DiffMessage(recovered, "> 0", "backtracking should recover paths the greedy walk misses"))
	}
}

func bruteForceMatch(routes []string, path string) (string, []string, bool) {
	pathSegs := strings.Split(strings.Trim(path, "/"), "/")
	best, bestParams := "", []string(nil)
	var bestRank []int
	ambiguous := false
	for _, route := range routes {
		routeSegs := strings.Split(strings.Trim(route, "/"), "/")
		if len(routeSegs) != len(pathSegs) {
			continue
		}
		rank := make([]int, len(routeSegs))
		var params []string
		matched := true
		for i, seg := range routeSegs {
			switch {
			case seg == pathSegs[i]:
				rank[i] = 0
			case seg == "$":
				rank[i] = 1
				params = append(params, pathSegs[i])
			case seg == "*":
				rank[i] = 2
			case strings.Contains(seg, "*") && matchWildcard(pathSegs[i], seg):
				rank[i] = 3
			default:
				matched = false
			}
			if !matched {
				break
			}
		}
		if !matched {
			continue
		}
		cmp := -1
		if bestRank != nil {
			cmp = 0
			for i := range rank {
				if rank[i] != bestRank[i] {
					cmp = 1
					if rank[i] < bestRank[i] {
						cmp = -1
					}
					break
				}
			}
		}
		if cmp < 0 {
			best, bestParams, bestRank, ambiguous = route, params, rank, false
		} else if cmp == 0 && route != best {
			ambiguous = true
		}
	}
	return best, bestParams, ambiguous
}

func randomRoutes(r *rand.Rand) []string {
	segs := []string{"a", "b", "c", "$", "$", "*", "x*"}
	routes := make([]string, 0, 8)
	for range 1 + r.IntN(8) {
		var b strings.Builder
		b.WriteByte('/')
		for range 1 + r.IntN(4) {
			b.WriteString(segs[r.IntN(len(segs))])
			b.WriteByte('/')
		}
		routes = append(routes, b.String())
	}
	return routes
}

func randomPath(r *rand.Rand) string {
	segs := []string{"a", "b", "c", "d", "xy", "x"}
	var b strings.Builder
	b.WriteByte('/')
	for range 1 + r.IntN(5) {
		b.WriteString(segs[r.IntN(len(segs))])
		b.WriteByte('/')
	}
	return b.String()
}

func findMaster(tr *Trie, path string, sep byte, supportParams bool) (string, string, []string) {
	node := tr
	var matchedNode *Trie
	var wildcardNode *Trie
	start := strings.IndexByte(path, sep)

	var paramVals []string

	for seg, next := str.Segment(path, sep, start); next > -1; seg, next = str.Segment(path, sep, next) {
		exactNode := node.Children[seg]
		if exactNode == nil {

			if supportParams && node.Children["$"] != nil {

				if w := node.Children["*"]; w != nil && w.IsEnd {
					wildcardNode = w
				}

				node = node.Children["$"]
				if paramVals == nil {
					paramVals = make([]string, 0, paramValsInitialCap)
				}
				paramVals = append(paramVals, seg)
			} else if node.Children["*"] != nil {
				if w := node.Children["*"]; w.IsEnd {
					wildcardNode = w
				}
				node = node.Children["*"]
			} else {
				isNotMatchAnythings := true

				for _, route := range node.globChildren {
					if matchWildcard(seg, route) {
						node = node.Children[route]
						isNotMatchAnythings = false
						break
					}
				}

				if isNotMatchAnythings {
					break
				}
			}
		} else {

			if w := node.Children["*"]; w != nil && w.IsEnd {
				wildcardNode = w
			}
			node = exactNode
		}

		if next == len(path)-1 {
			matchedNode = node

			if w := node.Children["*"]; w != nil && w.IsEnd {
				wildcardNode = w
			}
			break
		}

		continue
	}

	matchedRaw := ""
	if matchedNode != nil {
		matchedRaw = matchedNode.Raw
	}
	wildcardRaw := ""
	if wildcardNode != nil {
		wildcardRaw = wildcardNode.Raw
	}

	return matchedRaw, wildcardRaw, paramVals
}

var updateMatching = flag.Bool("update-matching", false, "rewrite MATCHING.md from the current Find behaviour")

type matchingCase struct {
	path string
	note string
}

type matchingScenario struct {
	title  string
	desc   string
	routes []string
	cases  []matchingCase
}

var matchingScenarios = []matchingScenario{
	{"1. Static routes", "Literal segments only; they must match exactly.", []string{"/health/", "/api/v1/users/"},
		[]matchingCase{{"/health/", ""}, {"/api/v1/users/", ""}, {"/api/v1/", "a prefix, not a route"}, {"/api/v1/users/extra/", "longer than the route"}, {"/Health/", "matching is case-sensitive"}}},
	{"2. Params (`{id}` is stored as `$`)", "`$` matches exactly one segment and captures it.", []string{"/users/$/", "/users/$/friends/$/"},
		[]matchingCase{{"/users/42/", ""}, {"/users/42/friends/7/", "two params"}, {"/users/42/friends/", "last segment missing"}, {"/users/", "param missing"}}},
	{"3. Priority: exact > param > `*` > glob", "When several children could match at the same position, the exact segment wins, then the param, then `*`, then globs.", []string{"/files/readme/", "/files/$/", "/files/*/", "/files/*.html/"},
		[]matchingCase{{"/files/readme/", "exact match"}, {"/files/abc/", "param beats `*` and glob"}, {"/files/index.html/", "param still beats the glob"}}},
	{"4. Wildcard `*` and globs", "`*` matches one segment, and a route ending in `*` is also the fallback for deeper paths. A glob such as `*.html` matches by pattern.", []string{"/static/*/", "/docs/*.html/", "/img/*.png/"},
		[]matchingCase{{"/static/app.js/", "`*` matches one segment"}, {"/static/js/app.js/", "deeper path: `/static/*/` is the fallback"}, {"/docs/index.html/", "glob"}, {"/docs/index.md/", "glob does not match"}, {"/img/logo.png/", ""}}},
	{"5. Backtracking", "The greedy walk takes the exact segment `me` and dead-ends; `Find` then backtracks and tries `$`.", []string{"/users/me/", "/users/$/profile/", "/users/$/"},
		[]matchingCase{{"/users/me/", "exact route"}, {"/users/me/profile/", "found by backtracking"}, {"/users/42/profile/", "normal match"}, {"/users/me/settings/", "no route fits"}}},
	{"6. Backtracking across levels", "A dead end deep in the trie, resolved one or two levels up.", []string{"/a/b/c/x/", "/a/$/c/y/", "/$/b/c/z/"},
		[]matchingCase{{"/a/b/c/x/", ""}, {"/a/b/c/y/", "backtracks at level 2 (`b` → `$`)"}, {"/a/b/c/z/", "backtracks at level 1 (`a` → `$`)"}, {"/a/b/c/w/", "no route fits"}}},
	{"7. A wildcard above the dead end wins", "If the greedy walk already has a wildcard fallback, `Find` does not backtrack. That keeps params consistent with the returned route.", []string{"/a/*/", "/a/b/x/", "/a/$/c/"},
		[]matchingCase{{"/a/b/c/", "the wildcard `/a/*/`, not `/a/$/c/`"}, {"/a/z/c/", "does not enter `b`, so `/a/$/c/` matches"}}},
	{"8. Params stay with their route on wildcard fallback", "The params returned always belong to the route the router will use.", []string{"/a/b/$/*/", "/a/$/c/d/e/"},
		[]matchingCase{{"/a/b/c/d/e/", "wildcard fallback with its own param `c`"}, {"/a/x/c/d/e/", "normal match"}}},
	{"9. Literal `$` or `*` in the request", "Request segments `$` and `*` are values, not placeholders: they take the param or wildcard branch.", []string{"/users/$/", "/files/*/"},
		[]matchingCase{{"/users/$/", "`$` is captured as the param"}, {"/files/*/", "takes the wildcard branch"}}},
	{"10. Missing trailing slash", "The trie expects enclosed paths (leading and trailing `/`). The router encloses request paths before calling `Find`.", []string{"/a/"},
		[]matchingCase{{"/a/", ""}, {"/a", "no trailing `/`"}}},
}

func TestMatchingTableIsUpToDate(t *testing.T) {
	want := renderMatchingTable()
	if *updateMatching {
		if err := os.WriteFile("MATCHING.md", []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile("MATCHING.md")
	if err != nil {
		t.Fatal(test.DiffMessage(err.Error(), "MATCHING.md present", "run with -update-matching to create it"))
	}
	if string(got) != want {
		t.Error(test.DiffMessage("MATCHING.md is stale", "MATCHING.md matches Find", "run go test ./internal/ds -run TestMatchingTableIsUpToDate -update-matching"))
	}
}

func renderMatchingTable() string {
	var b strings.Builder
	b.WriteString("# Route matching in `internal/ds` (Trie.Find)\n\n")
	b.WriteString("Every row below is the actual output of `Trie.Find`, generated by `TestMatchingTableIsUpToDate` in `trie_test.go`. The test fails if this file drifts from the code. To regenerate it, run `go test ./internal/ds -run TestMatchingTableIsUpToDate -update-matching`.\n\n")
	b.WriteString("- **Result** is what the router will use: the matched route, or else the wildcard fallback.\n")
	b.WriteString("- Paths are given as the router passes them to the trie, starting and ending with `/`. A `{name}` param in a route is stored as `$`.\n\n")
	for _, sc := range matchingScenarios {
		tr := NewTrie()
		for _, route := range sc.routes {
			tr.Insert(route, route, '/')
		}
		b.WriteString("## " + sc.title + "\n\n" + sc.desc + "\n\n")
		b.WriteString("Registered routes: `" + strings.Join(sc.routes, "`, `") + "`\n\n")
		b.WriteString("| Request path | Result | Params | Note |\n|---|---|---|---|\n")
		for _, c := range sc.cases {
			raw, wildcard, params := tr.Find(c.path, '/')
			result := "❌ no match"
			switch {
			case raw != "":
				result = "✅ `" + raw + "` (route)"
			case wildcard != "":
				result = "✅ `" + wildcard + "` (wildcard fallback)"
			}
			paramCell := "—"
			if len(params) > 0 && result != "❌ no match" {
				paramCell = "`" + strings.Join(params, "`, `") + "`"
			}
			note := c.note
			if note == "" {
				note = "—"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s |\n", c.path, result, paramCell, note)
		}
		b.WriteString("\n")
	}
	return b.String()
}
