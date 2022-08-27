package ds

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

func TestAddAtBegin(test *testing.T) {
	expect1 := "_foo/bar/baz/"
	output1 := AddAtBegin("foo/bar/baz/", UNDERSCORE)
	if output1 != expect1 {
		test.Errorf("AddAtBegin(\"foo/bar/baz/\", UNDERSCORE) = %v; expect = %v", output1, expect1)
	}

	unexpect2 := "**foo/bar/baz/"
	output2 := AddAtBegin("*foo/bar/baz/", WILDCARD)
	if output2 == unexpect2 {
		test.Errorf("AddAtBegin(\"*foo/bar/baz/\", WILDCARD) = %v; expect ≠ %v", output2, unexpect2)
	}
}

func TestRemoveAtBegin(test *testing.T) {
	expect1 := "foo/bar/baz"
	output1 := RemoveAtBegin("{foo/bar/baz", OPEN_CURLY_BRACKET)
	if output1 != expect1 {
		test.Errorf("RemoveAtBegin(\"{foo/bar/baz\", OPEN_CURLY_BRACKET) = %v; expect = %v", output1, expect1)
	}

	expect2 := "foo/*/bar/baz/"
	output2 := RemoveAtBegin("/*/foo/*/bar/baz/", SLASH+WILDCARD+SLASH)
	if output2 != expect2 {
		test.Errorf("RemoveAtBegin(\"foo/*/bar/baz/\", SLASH+WILDCARD+SLASH) = %v; expect = %v", output2, expect2)
	}
}

func TestAddAtEnd(test *testing.T) {
	expect1 := "/foo/bar/baz/{}"
	output1 := AddAtEnd("/foo/bar/baz/", OPEN_CURLY_BRACKET+CLOSE_CURLY_BRACKET)
	if output1 != expect1 {
		test.Errorf("AddAtEnd(\"/foo/bar/baz/\", OPEN_CURLY_BRACKET+CLOSE_CURLY_BRACKET) = %v; expect = %v", output1, expect1)
	}

	unexpect2 := "/foo/bar/baz/****"
	output2 := AddAtEnd("/foo/bar/baz/**", WILDCARD+WILDCARD)
	if output2 == unexpect2 {
		test.Errorf("AddAtEnd(\"/foo/bar/baz/**\", WILDCARD+WILDCARD) = %v; expect = %v", output2, unexpect2)
	}
}

func TestRemoveAtEnd(test *testing.T) {
	expect1 := "/foo/{}bar/baz/"
	output1 := RemoveAtEnd("/foo/{}bar/baz/{}", OPEN_CURLY_BRACKET+CLOSE_CURLY_BRACKET)
	if output1 != expect1 {
		test.Errorf("RemoveAtEnd(\"/foo/{}bar/baz/{}\", OPEN_CURLY_BRACKET+CLOSE_CURLY_BRACKET) = %v; expect = %v", output1, expect1)
	}

	expect2 := "/foo/*bar/baz"
	output2 := RemoveAtEnd("/foo/*bar/baz///", SLASH+SLASH+SLASH)
	if output2 != expect2 {
		test.Errorf("RemoveAtEnd(\"/foo/*bar/baz///\", SLASH+SLASH+SLASH) = %v; expect = %v", output2, expect2)
	}
}
