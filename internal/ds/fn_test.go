package ds

import (
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

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
		{"a.html.html", "*.html", true},
		{"aaaa", "*a", true},
		{"a", "*", true},
		{"", "*", true},
		{"abc", "**", true},
		{"x.y.z", "*.z", true},
		{"x.y.z", "x.*", true},
	}

	for _, c := range cases {
		got := matchWildcard(c.str, c.route)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "matchWildcard("+c.str+", "+c.route+")"))
		}
	}
}
