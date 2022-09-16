package utils

import (
	"strings"
	"testing"
)

func BenchmarkStrSegment(t *testing.B) {
	input1 := "/users/{userId}/schools/{schoolId}/subjects/{subjectId}/"
	sep := byte('/')
	start := strings.IndexByte(input1, sep)
	for i := 0; i < t.N; i++ {
		for _, next := StrSegment(input1, sep, start); next >= 0; _, next = StrSegment(input1, sep, next) {

		}
	}
}
