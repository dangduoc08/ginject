package str

import (
	"strings"
	"testing"
)

func BenchmarkSegment(t *testing.B) {
	input1 := "/users/{userId}/schools/{schoolId}/subjects/{subjectId}/"
	sep := byte('/')
	start := strings.IndexByte(input1, sep)
	for i := 0; i < t.N; i++ {
		for _, next := Segment(input1, sep, start); next >= 0; _, next = Segment(input1, sep, next) {

		}
	}
}

func BenchmarkRemoveSpace(b *testing.B) {
	input := "A B C D E F G H I J K L M N O P Q R S T U V W X Y Z a b c d e f g h i j k l m n o p q r s t u v w x y z"
	for i := 0; i < b.N; i++ {
		RemoveSpace(input)
	}
}

func BenchmarkRemoveDup(b *testing.B) {
	input := "/**/school**/***/***/{subjectId}/***"
	for i := 0; i < b.N; i++ {
		RemoveDup(input, "*")
	}
}

func BenchmarkIsLower(b *testing.B) {
	input := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < b.N; i++ {
		IsLower(input)
	}
}

func BenchmarkIsUpper(b *testing.B) {
	input := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < b.N; i++ {
		IsUpper(input)
	}
}
