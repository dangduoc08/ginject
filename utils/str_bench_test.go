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

func BenchmarkStrRemoveSpace(b *testing.B) {
	input := "A B C D E F G H I J K L M N O P Q R S T U V W X Y Z a b c d e f g h i j k l m n o p q r s t u v w x y z"
	for i := 0; i < b.N; i++ {
		StrRemoveSpace(input)
	}
}

func BenchmarkStrRemoveDup(b *testing.B) {
	input := "/**/school**/***/***/{subjectId}/***"
	for i := 0; i < b.N; i++ {
		StrRemoveDup(input, "*")
	}
}

func BenchmarkStrIsLower(b *testing.B) {
	input := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < b.N; i++ {
		StrIsLower(input)
	}
}

func BenchmarkStrIsUpper(b *testing.B) {
	input := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	for i := 0; i < b.N; i++ {
		StrIsUpper(input)
	}
}

func BenchmarkStrWithCharset(b *testing.B) {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for i := 0; i < b.N; i++ {
		StrWithCharset(32, charset)
	}
}
