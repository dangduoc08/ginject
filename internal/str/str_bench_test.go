package str

import (
	"strings"
	"testing"
)

func BenchmarkSegment(b *testing.B) {
	b.ReportAllocs()
	input1 := "/users/{userId}/schools/{schoolId}/subjects/{subjectId}/"
	sep := byte('/')
	start := strings.IndexByte(input1, sep)
	for b.Loop() {
		for _, next := Segment(input1, sep, start); next >= 0; _, next = Segment(input1, sep, next) {

		}
	}
}

func BenchmarkRemoveSpace(b *testing.B) {
	b.ReportAllocs()
	input := "A B C D E F G H I J K L M N O P Q R S T U V W X Y Z a b c d e f g h i j k l m n o p q r s t u v w x y z"
	for b.Loop() {
		RemoveSpace(input)
	}
}

func BenchmarkEnclose(b *testing.B) {
	b.ReportAllocs()
	input := "//users//profile//settings//"
	for b.Loop() {
		Enclose(input, '/')
	}
}

func BenchmarkRemoveDup(b *testing.B) {
	b.ReportAllocs()
	input := "/**/school**/***/***/{subjectId}/***"
	for b.Loop() {
		RemoveDup(input, "*")
	}
}

func BenchmarkEnclose_RequestPath(b *testing.B) {
	for _, in := range []string{"/api/v1/users/123/orders", "/api/v1/users/123/orders/"} {
		b.Run(in, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				Enclose(in, '/')
			}
		})
	}
}
