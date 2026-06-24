package routing

import (
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestPatternToMethodRouteVersion(t *testing.T) {
	cases := []struct {
		pattern     string
		wantMethod  string
		wantRoute   string
		wantVersion string
	}{
		{
			pattern:     "/users/$/||/[GET]/",
			wantMethod:  "GET",
			wantRoute:   "/users/$",
			wantVersion: "",
		},
		{
			pattern:     "/users/$/|v2|/[POST]/",
			wantMethod:  "POST",
			wantRoute:   "/users/$",
			wantVersion: "v2",
		},
		{
			pattern:     "/feeds/all/||/[DELETE]/",
			wantMethod:  "DELETE",
			wantRoute:   "/feeds/all",
			wantVersion: "",
		},
	}

	for _, c := range cases {
		method, route, version := PatternToMethodRouteVersion(c.pattern)
		if method != c.wantMethod {
			t.Error(test.DiffMessage(method, c.wantMethod, "PatternToMethodRouteVersion method"))
		}
		if route != c.wantRoute {
			t.Error(test.DiffMessage(route, c.wantRoute, "PatternToMethodRouteVersion route"))
		}
		if version != c.wantVersion {
			t.Error(test.DiffMessage(version, c.wantVersion, "PatternToMethodRouteVersion version"))
		}
	}
}

func TestToEndpoint(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"users", "/users/"},
		{"/users/", "/users/"},
		{"//users//", "/users/"},
		{" /users/ ", "/users/"},
		{"/a//b///c/", "/a/b/c/"},
		{"/a/**/b/", "/a/*/b/"},
	}

	for _, c := range cases {
		got := ToEndpoint(c.in)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "ToEndpoint"))
		}
	}
}

func TestParseToParamKey(t *testing.T) {
	str, keys := ParseToParamKey("/users/{userId}/friends/{friendId}/")
	wantStr := "/users/$/friends/$/"
	if str != wantStr {
		t.Error(test.DiffMessage(str, wantStr, "ParseToParamKey str"))
	}
	if keys["userId"][0] != 0 {
		t.Error(test.DiffMessage(keys["userId"][0], 0, "ParseToParamKey userId index"))
	}
	if keys["friendId"][0] != 1 {
		t.Error(test.DiffMessage(keys["friendId"][0], 1, "ParseToParamKey friendId index"))
	}

	str2, keys2 := ParseToParamKey("/plain/route/")
	if str2 != "/plain/route/" {
		t.Error(test.DiffMessage(str2, "/plain/route/", "ParseToParamKey no params"))
	}
	if len(keys2) != 0 {
		t.Error(test.DiffMessage(len(keys2), 0, "ParseToParamKey no param keys"))
	}
}

func TestMatchWildcard(t *testing.T) {
	cases := []struct {
		str   string
		route string
		want  bool
	}{
		{"index.html", "*.html", true},
		{"image.png", "image.*", true},
		{"index.html", "in*.html", true},
		{"in.html", "in*.html", true},
		{"index.html", "*.png", false},
		{"foo", "foo", true},
		{"foobar", "foo", false},
	}

	for _, c := range cases {
		got := matchWildcard(c.str, c.route)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "matchWildcard("+c.str+", "+c.route+")"))
		}
	}
}

func TestFromMethodtoPattern(t *testing.T) {
	got := toPattern(http.MethodGet, "[", "]")
	want := "[GET]"
	if got != want {
		t.Error(test.DiffMessage(got, want, "fromMethodtoPattern"))
	}
}

func TestFromVersiontoPattern(t *testing.T) {
	if got := toPattern("", "|", "|"); got != "||" {
		t.Error(test.DiffMessage(got, "||", "toPattern empty"))
	}
	if got := toPattern("v2", "|", "|"); got != "|v2|" {
		t.Error(test.DiffMessage(got, "|v2|", "toPattern v2"))
	}
}

func TestToPattern(t *testing.T) {
	cases := []struct {
		s, l, r, want string
	}{
		{"", "[", "]", "[]"},
		{"GET", "[", "]", "[GET]"},
		{"[GET]", "[", "]", "[GET]"},
		{"[GET", "[", "]", "[GET]"},
		{"GET]", "[", "]", "[GET]"},
		{"  GET  ", "[", "]", "[GET]"},
	}

	for _, c := range cases {
		got := toPattern(c.s, c.l, c.r)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "toPattern("+c.s+", "+c.l+", "+c.r+")"))
		}
	}
}

func TestMethodRouteVersionToPattern(t *testing.T) {
	cases := []struct {
		method, route, version, want string
	}{
		{http.MethodGet, "/users/{userId}", "", "/users/{userId}/||/[GET]/"},
		{http.MethodPost, "/users/{userId}", "v2", "/users/{userId}/|v2|/[POST]/"},
		{http.MethodDelete, "/feeds/all", "", "/feeds/all/||/[DELETE]/"},
		{"", "/feeds/all", "", "/feeds/all/||/[]/"},
	}

	for _, c := range cases {
		got := MethodRouteVersionToPattern(c.method, c.route, c.version)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "MethodRouteVersionToPattern("+c.method+", "+c.route+", "+c.version+")"))
		}
	}
}

func TestResolveWildcardRoute(t *testing.T) {
	versionPattern := "|v2|"
	methodPattern := "{method}"

	cases := []struct {
		name        string
		insertedStr string
		wantIndex   int
	}{
		{"terminal at *", "/abc/xyz/*/", 10},
		{"full depth */version/method", "/abc/xyz/*/|v2|/{method}/", 11},
		{"terminal at version, no method", "/abc/xyz/*/|v2|/", 12},
		{"method directly under *, no version", "/abc/xyz/*/{method}/", 13},
	}

	for _, c := range cases {
		tr := NewTrie()
		tr.insert(c.insertedStr, c.insertedStr, '/', c.wantIndex)

		xyzNode := tr.Children["abc"].Children["xyz"]
		got := resolveWildcardRoute(xyzNode, versionPattern, methodPattern)

		if got == nil {
			t.Error(test.DiffMessage(got, c.wantIndex, c.name+": expected a resolved node"))
			continue
		}
		if got.Index != c.wantIndex {
			t.Error(test.DiffMessage(got.Index, c.wantIndex, c.name+": resolved node Index"))
		}
	}

	noMatch := NewTrie()
	noMatch.insert("/abc/xyz/sibling/", "/abc/xyz/sibling/", '/', 99)
	xyzNode := noMatch.Children["abc"].Children["xyz"]
	if got := resolveWildcardRoute(xyzNode, versionPattern, methodPattern); got != nil {
		t.Error(test.DiffMessage(got.Index, -1, "no * / version / method reachable: expected nil"))
	}
}
