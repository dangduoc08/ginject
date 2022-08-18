package helper

import (
	"testing"
)

func TestRemoveSpace(test *testing.T) {
	r1 := RemoveSpace("A B CDE")
	expect := "ABCDE"
	if r1 != expect {
		test.Errorf("RemoveSpace(\"A B CDE\") = %v; expect = %v", r1, expect)
	}
}

func TestAddSlash(test *testing.T) {
	expect := "/foo/bar/baz/"

	r1 := AddSlash("foo/bar/baz")
	if r1 != expect {
		test.Errorf("AddSlash(\"/foo/bar/baz\") = %v; expect = %v", r1, expect)
	}

	r2 := AddSlash("/foo/bar/baz/")
	if r2 != expect {
		test.Errorf("AddSlash(\"/foo/bar/baz/\") = %v; expect = %v", r2, expect)
	}
}
