package test

import (
	"strings"
	"testing"
)

func TestDiffMessage_Layout(t *testing.T) {
	cases := []struct {
		name     string
		actual   any
		expected any
		desc     string
		want     string
	}{
		{"with desc", 2, 3, "sum", "sum\nexpected: 3\nactual:   2"},
		{"no desc", 2, 3, "", "\nexpected: 3\nactual:   2"},
		{"int vs string", 1, "1", "", "\nexpected: \"1\"\nactual:   1"},
		{"empty vs space", "", " ", "", "\nexpected: \" \"\nactual:   \"\""},
		{"empty vs nil slice", []string{}, []string(nil), "", "\nexpected: []string(nil)\nactual:   []string{}"},
		{"untyped nil", nil, nil, "", "\nexpected: <nil>\nactual:   <nil>"},
		{"struct", struct{ A int }{1}, struct{ A int }{2}, "", "\nexpected: struct { A int }{A:2}\nactual:   struct { A int }{A:1}"},
	}

	for _, c := range cases {
		got := stripANSI(DiffMessage(c.actual, c.expected, c.desc))
		if got != c.want {
			t.Errorf("%s:\n got: %s\nwant: %s", c.name, got, c.want)
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func TestDiffMessage_Colors(t *testing.T) {
	got := DiffMessage(2, 3, "sum")
	want := "\x1b[1msum\x1b[22m\n\x1b[2mexpected:\x1b[22m \x1b[32m3\x1b[39m\n\x1b[2mactual:\x1b[22m   \x1b[31m2\x1b[39m"
	if got != want {
		t.Errorf("DiffMessage colors:\n got: %q\nwant: %q", got, want)
	}
}
