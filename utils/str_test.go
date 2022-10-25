package utils

import (
	"strings"
	"testing"
)

func TestStrRemoveSpace(t *testing.T) {
	output1 := StrRemoveSpace("A B CDE")
	expect := "ABCDE"
	if output1 != expect {
		t.Errorf("StrRemoveSpace(\"A B CDE\") = %v; expect = %v", output1, expect)
	}
}

func TestStrAddBegin(t *testing.T) {
	expect1 := "_foo/bar/baz/"
	output1 := StrAddBegin("foo/bar/baz/", "_")
	if output1 != expect1 {
		t.Errorf("StrAddBegin(\"foo/bar/baz/\", _) = %v; expect = %v", output1, expect1)
	}

	unexpect2 := "**foo/bar/baz/"
	output2 := StrAddBegin("*foo/bar/baz/", "*")
	if output2 == unexpect2 {
		t.Errorf("StrAddBegin(\"*foo/bar/baz/\", *) = %v; expect ≠ %v", output2, unexpect2)
	}
}

func TestStrRemoveBegin(t *testing.T) {
	expect1 := "foo/bar/baz"
	output1 := StrRemoveBegin("{foo/bar/baz", "{")
	if output1 != expect1 {
		t.Errorf("StrRemoveBegin(\"{foo/bar/baz\", {) = %v; expect = %v", output1, expect1)
	}

	expect2 := "foo/*/bar/baz/"
	output2 := StrRemoveBegin("/*/foo/*/bar/baz/", "/*/")
	if output2 != expect2 {
		t.Errorf("StrRemoveBegin(\"foo/*/bar/baz/\", /*/) = %v; expect = %v", output2, expect2)
	}
}

func TestStrAddEnd(t *testing.T) {
	expect1 := "/foo/bar/baz/{}"
	output1 := StrAddEnd("/foo/bar/baz/", "{}")
	if output1 != expect1 {
		t.Errorf("StrAddEnd(\"/foo/bar/baz/\", {}) = %v; expect = %v", output1, expect1)
	}

	unexpect2 := "/foo/bar/baz/****"
	output2 := StrAddEnd("/foo/bar/baz/**", "**")
	if output2 == unexpect2 {
		t.Errorf("StrAddEnd(\"/foo/bar/baz/**\", **) = %v; expect = %v", output2, unexpect2)
	}
}

func TestStrRemoveEnd(t *testing.T) {
	expect1 := "/foo/{}bar/baz/"
	output1 := StrRemoveEnd("/foo/{}bar/baz/{}", "{}")
	if output1 != expect1 {
		t.Errorf("StrRemoveEnd(\"/foo/{}bar/baz/{}\", {}) = %v; expect = %v", output1, expect1)
	}

	expect2 := "/foo/*bar/baz"
	output2 := StrRemoveEnd("/foo/*bar/baz///", "///")
	if output2 != expect2 {
		t.Errorf("StrRemoveEnd(\"/foo/*bar/baz///\", ///) = %v; expect = %v", output2, expect2)
	}
}

func TestStrSegment(t *testing.T) {
	input1 := "/users/{userId}/schools/{schoolId}/subjects/{subjectId}/"
	expect1 := make([]string, 6)
	i := -1
	for seg, next := StrSegment(input1, '/', 0); next >= 0; seg, next = StrSegment(input1, '/', next) {
		i++
		expect1[i] = seg
	}

	spl := strings.Split(input1, "/")
	for i, seg := range expect1 {
		if seg != spl[i+1] {
			t.Errorf("StrSegment return seg = %v; expect = %v", seg, spl[i+1])
		}
	}
}

func TestStrRemoveDup(t *testing.T) {
	expect1 := "/*/school*/*/*/{subjectId}/*"
	output1 := StrRemoveDup("/**/school**/***/***/{subjectId}/***", "*")

	if expect1 != output1 {
		t.Errorf("StrRemoveDup(\"/**/school**/***/***/{subjectId}/***, *\") = %v; expect = %v", output1, expect1)
	}
}

func TestStrIsLower(t *testing.T) {
	expect1 := true
	output1 := StrIsLower("foo")[0]

	if expect1 != output1 {
		t.Errorf("StrIsLower(\"foo\") = %v; expect = %v", output1, expect1)
	}

	expect2 := false
	output2 := StrIsLower("Baz")[0]

	if expect2 != output2 {
		t.Errorf("StrIsLower(\"Baz\") = %v; expect = %v", output2, expect2)
	}
}

func TestStrIsUpper(t *testing.T) {
	expect1 := true
	output1 := StrIsUpper("foO")[2]

	if expect1 != output1 {
		t.Errorf("StrIsUpper(\"foO\") = %v; expect = %v", output1, expect1)
	}

	expect2 := false
	output2 := StrIsUpper("baZ")[0]

	if expect2 != output2 {
		t.Errorf("StrIsUpper(\"baZ\") = %v; expect = %v", output2, expect2)
	}
}
