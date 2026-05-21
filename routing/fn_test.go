package routing

import (
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/testutils"
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
			t.Error(testutils.DiffMessage(method, c.wantMethod, "PatternToMethodRouteVersion method"))
		}
		if route != c.wantRoute {
			t.Error(testutils.DiffMessage(route, c.wantRoute, "PatternToMethodRouteVersion route"))
		}
		if version != c.wantVersion {
			t.Error(testutils.DiffMessage(version, c.wantVersion, "PatternToMethodRouteVersion version"))
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
			t.Error(testutils.DiffMessage(got, c.want, "ToEndpoint"))
		}
	}
}

func TestParseToParamKey(t *testing.T) {
	str, keys := ParseToParamKey("/users/{userId}/friends/{friendId}/")
	wantStr := "/users/$/friends/$/"
	if str != wantStr {
		t.Error(testutils.DiffMessage(str, wantStr, "ParseToParamKey str"))
	}
	if keys["userId"][0] != 0 {
		t.Error(testutils.DiffMessage(keys["userId"][0], 0, "ParseToParamKey userId index"))
	}
	if keys["friendId"][0] != 1 {
		t.Error(testutils.DiffMessage(keys["friendId"][0], 1, "ParseToParamKey friendId index"))
	}

	str2, keys2 := ParseToParamKey("/plain/route/")
	if str2 != "/plain/route/" {
		t.Error(testutils.DiffMessage(str2, "/plain/route/", "ParseToParamKey no params"))
	}
	if len(keys2) != 0 {
		t.Error(testutils.DiffMessage(len(keys2), 0, "ParseToParamKey no param keys"))
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
			t.Error(testutils.DiffMessage(got, c.want, "matchWildcard("+c.str+", "+c.route+")"))
		}
	}
}

func TestFromMethodtoPattern(t *testing.T) {
	got := fromMethodtoPattern(http.MethodGet)
	want := "[GET]"
	if got != want {
		t.Error(testutils.DiffMessage(got, want, "fromMethodtoPattern"))
	}
}

func TestFromVersiontoPattern(t *testing.T) {
	if got := fromVersiontoPattern(""); got != "||" {
		t.Error(testutils.DiffMessage(got, "||", "fromVersiontoPattern empty"))
	}
	if got := fromVersiontoPattern("v2"); got != "|v2|" {
		t.Error(testutils.DiffMessage(got, "|v2|", "fromVersiontoPattern v2"))
	}
}
