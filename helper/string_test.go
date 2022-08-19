package helper

import (
	"testing"
)

func TestRemoveSpace(test *testing.T) {
	output1 := RemoveSpace("A B CDE")
	expect := "ABCDE"
	if output1 != expect {
		test.Errorf("RemoveSpace(\"A B CDE\") = %v; expect = %v", output1, expect)
	}
}

func TestAddFirstSlash(test *testing.T) {
	expect := "/foo/bar/baz/"

	output1 := AddFirstSlash("foo/bar/baz/")
	if output1 != expect {
		test.Errorf("AddFirstSlash(\"foo/bar/baz/\") = %v; expect = %v", output1, expect)
	}
}

func TestAddLastSlash(test *testing.T) {
	expect := "/foo/bar/baz/"

	output1 := AddLastSlash("/foo/bar/baz")
	if output1 != expect {
		test.Errorf("AddLastSlash(\"/foo/bar/baz\") = %v; expect = %v", output1, expect)
	}
}

func TestRemoveLastSlash(test *testing.T) {
	expect := "/foo/bar/baz"

	output1 := RemoveLastSlash("/foo/bar/baz/")
	if output1 != expect {
		test.Errorf("RemoveLastSlash(\"/foo/bar/baz/\") = %v; expect = %v", output1, expect)
	}
}

func TestRemoveFirstColon(test *testing.T) {
	expect := "foo/bar/baz/"

	output1 := RemoveFirstColon(":foo/bar/baz/")
	if output1 != expect {
		test.Errorf("RemoveFirstColon(\":foo/bar/baz/\") = %v; expect = %v", output1, expect)
	}
}
