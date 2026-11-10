package str

import (
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestRemoveSpace(t *testing.T) {
	output1 := RemoveSpace("A B CDE")
	expect := "ABCDE"
	if output1 != expect {
		t.Error(test.DiffMessage(output1, expect, "RemoveSpace"))
	}
}

func TestSegment(t *testing.T) {
	input1 := "/users/{userId}/schools/{schoolId}/subjects/{subjectId}/"
	expect1 := make([]string, 6)
	i := -1
	for seg, next := Segment(input1, '/', 0); next >= 0; seg, next = Segment(input1, '/', next) {
		i++
		expect1[i] = seg
	}

	spl := strings.Split(input1, "/")
	for i, seg := range expect1 {
		if seg != spl[i+1] {
			t.Error(test.DiffMessage(seg, spl[i+1], "Segment"))
		}
	}
}

func TestEnclose(t *testing.T) {
	cases := []struct {
		in   string
		sep  byte
		want string
	}{
		{"users", '/', "/users/"},
		{"/users/", '/', "/users/"},
		{"//users//", '/', "/users/"},
		{" /users/ ", '/', "/users/"},
		{"/a//b///c/", '/', "/a/b/c/"},
		{"/a/**/b/", '/', "/a/*/b/"},
		{"foo", '.', ".foo."},
		{"foo.bar", '.', ".foo.bar."},
		{".foo.bar.", '.', ".foo.bar."},
		{"foo..bar", '.', ".foo.bar."},
		{"", '.', ""},
		{"/", '/', "/"},
		{"   ", '/', ""},
		{"/us er/", '/', "/user/"},
		{"/x/\t/y", '/', "/x/y/"},
		{"/a***b/", '/', "/a*b/"},
	}

	for _, c := range cases {
		got := Enclose(c.in, c.sep)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "Enclose("+c.in+")"))
		}
	}
}

func TestRemoveDup(t *testing.T) {
	expect1 := "/*/school*/*/*/{subjectId}/*"
	output1 := RemoveDup("/**/school**/***/***/{subjectId}/***", "*")
	if expect1 != output1 {
		t.Error(test.DiffMessage(output1, expect1, "RemoveDup"))
	}
}

func TestSegment_Edges(t *testing.T) {
	cases := []struct {
		in       string
		start    int
		wantSeg  string
		wantNext int
	}{
		{"/a/b/", 0, "a", 2},
		{"/a/b/", 2, "b", 4},
		{"/a/b/", 4, "/", -1},
		{"/a/b", 2, "/b", -1},
		{"a/b/", 0, "", 1},
		{"", 0, "", -1},
		{"/a/", -1, "", -1},
		{"/a/", 3, "", -1},
	}

	for _, c := range cases {
		seg, next := Segment(c.in, '/', c.start)
		if seg != c.wantSeg || next != c.wantNext {
			t.Error(test.DiffMessage([]any{seg, next}, []any{c.wantSeg, c.wantNext}, "Segment("+c.in+")"))
		}
	}
}

func TestRemoveDup_Edges(t *testing.T) {
	cases := []struct {
		in      string
		pattern string
		want    string
	}{
		{"//a//b//", "/", "/a/b/"},
		{"aaa", "a", "a"},
		{"", "/", ""},
		{"a//b", "//", "a//b"},
		{"a//b", "", "a//b"},
		{"xééy", "é", "xéy"},
	}

	for _, c := range cases {
		if got := RemoveDup(c.in, c.pattern); got != c.want {
			t.Error(test.DiffMessage(got, c.want, "RemoveDup("+c.in+", "+c.pattern+")"))
		}
	}
}

func TestRemoveSpace_Unicode(t *testing.T) {
	got := RemoveSpace("a\tb\nc\u00a0d\u2003e f")
	if got != "abcdef" {
		t.Error(test.DiffMessage(got, "abcdef", "RemoveSpace unicode spaces"))
	}
}

func TestEnclose_FastPathMatchesSlowPath(t *testing.T) {
	alphabet := []byte{'/', '/', '.', 'a', 'b', '*', ' ', '\t', '{', '}'}
	r := rand.New(rand.NewPCG(1, 2))
	for range 200000 {
		b := make([]byte, r.IntN(12))
		for i := range b {
			b[i] = alphabet[r.IntN(len(alphabet))]
		}
		in := string(b)
		for _, sep := range []byte{'/', '.'} {
			if got, want := Enclose(in, sep), encloseSlow(in, sep); got != want {
				t.Fatal(test.DiffMessage(got, want, "Enclose fast path vs slow path for "+in))
			}
		}
	}
}

func TestEnclose_FastPathReturnsInputWhenAlreadyEnclosed(t *testing.T) {
	in := "/api/v1/users/123/orders/"
	if got := Enclose(in, '/'); got != in {
		t.Error(test.DiffMessage(got, in, "already enclosed path"))
	}
	if n := testing.AllocsPerRun(100, func() { Enclose(in, '/') }); n != 0 {
		t.Error(test.DiffMessage(n, 0.0, "allocs for an already enclosed path"))
	}
}
