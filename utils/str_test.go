package utils

import (
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/testutils"
)

func TestStrRemoveSpace(t *testing.T) {
	output1 := StrRemoveSpace("A B CDE")
	expect := "ABCDE"
	if output1 != expect {
		t.Error(testutils.DiffMessage(output1, expect, "StrRemoveSpace"))
	}
}

func TestStrAddBegin(t *testing.T) {
	expect1 := "_foo/bar/baz/"
	output1 := StrAddBegin("foo/bar/baz/", "_")
	if output1 != expect1 {
		t.Error(testutils.DiffMessage(output1, expect1, "StrAddBegin"))
	}

	unexpect2 := "**foo/bar/baz/"
	output2 := StrAddBegin("*foo/bar/baz/", "*")
	if output2 == unexpect2 {
		t.Error(testutils.DiffMessage(output2, "≠ "+unexpect2, "StrAddBegin should not double prefix"))
	}
}

func TestStrAddEnd(t *testing.T) {
	expect1 := "/foo/bar/baz/{}"
	output1 := StrAddEnd("/foo/bar/baz/", "{}")
	if output1 != expect1 {
		t.Error(testutils.DiffMessage(output1, expect1, "StrAddEnd"))
	}

	unexpect2 := "/foo/bar/baz/****"
	output2 := StrAddEnd("/foo/bar/baz/**", "**")
	if output2 == unexpect2 {
		t.Error(testutils.DiffMessage(output2, "≠ "+unexpect2, "StrAddEnd should not double suffix"))
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
			t.Error(testutils.DiffMessage(seg, spl[i+1], "StrSegment"))
		}
	}
}

func TestStrRemoveDup(t *testing.T) {
	expect1 := "/*/school*/*/*/{subjectId}/*"
	output1 := StrRemoveDup("/**/school**/***/***/{subjectId}/***", "*")
	if expect1 != output1 {
		t.Error(testutils.DiffMessage(output1, expect1, "StrRemoveDup"))
	}
}

func TestStrIsLower(t *testing.T) {
	output1 := StrIsLower("foo")[0]
	if output1 != true {
		t.Error(testutils.DiffMessage(output1, true, "StrIsLower"))
	}

	output2 := StrIsLower("Baz")[0]
	if output2 != false {
		t.Error(testutils.DiffMessage(output2, false, "StrIsLower"))
	}
}

func TestStrIsUpper(t *testing.T) {
	output1 := StrIsUpper("foO")[2]
	if output1 != true {
		t.Error(testutils.DiffMessage(output1, true, "StrIsUpper"))
	}

	output2 := StrIsUpper("baZ")[0]
	if output2 != false {
		t.Error(testutils.DiffMessage(output2, false, "StrIsUpper"))
	}
}
