package str

import (
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

func TestAddBegin(t *testing.T) {
	expect1 := "_foo/bar/baz/"
	output1 := AddBegin("foo/bar/baz/", "_")
	if output1 != expect1 {
		t.Error(test.DiffMessage(output1, expect1, "AddBegin"))
	}

	unexpect2 := "**foo/bar/baz/"
	output2 := AddBegin("*foo/bar/baz/", "*")
	if output2 == unexpect2 {
		t.Error(test.DiffMessage(output2, "≠ "+unexpect2, "AddBegin should not double prefix"))
	}
}

func TestAddEnd(t *testing.T) {
	expect1 := "/foo/bar/baz/{}"
	output1 := AddEnd("/foo/bar/baz/", "{}")
	if output1 != expect1 {
		t.Error(test.DiffMessage(output1, expect1, "AddEnd"))
	}

	unexpect2 := "/foo/bar/baz/****"
	output2 := AddEnd("/foo/bar/baz/**", "**")
	if output2 == unexpect2 {
		t.Error(test.DiffMessage(output2, "≠ "+unexpect2, "AddEnd should not double suffix"))
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

func TestRemoveDup(t *testing.T) {
	expect1 := "/*/school*/*/*/{subjectId}/*"
	output1 := RemoveDup("/**/school**/***/***/{subjectId}/***", "*")
	if expect1 != output1 {
		t.Error(test.DiffMessage(output1, expect1, "RemoveDup"))
	}
}

func TestIsLower(t *testing.T) {
	output1 := IsLower("foo")[0]
	if output1 != true {
		t.Error(test.DiffMessage(output1, true, "IsLower"))
	}

	output2 := IsLower("Baz")[0]
	if output2 != false {
		t.Error(test.DiffMessage(output2, false, "IsLower"))
	}
}

func TestIsUpper(t *testing.T) {
	output1 := IsUpper("foO")[2]
	if output1 != true {
		t.Error(test.DiffMessage(output1, true, "IsUpper"))
	}

	output2 := IsUpper("baZ")[0]
	if output2 != false {
		t.Error(test.DiffMessage(output2, false, "IsUpper"))
	}
}
