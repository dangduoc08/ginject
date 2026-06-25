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
