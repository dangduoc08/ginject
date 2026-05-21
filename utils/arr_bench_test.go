package utils

import "testing"

var benchStrings = func() []string {
	s := make([]string, 1000)
	for i := range s {
		if i%2 == 0 {
			s[i] = "keep"
		} else {
			s[i] = "drop"
		}
	}
	return s
}()

var benchInts = func() []int {
	s := make([]int, 1000)
	for i := range s {
		s[i] = i
	}
	return s
}()

var benchDups = func() []int {
	s := make([]int, 1000)
	for i := range s {
		s[i] = i % 100
	}
	return s
}()

var benchNumStrs = func() []string {
	s := make([]string, 1000)
	for i := range s {
		s[i] = "42"
	}
	return s
}()

func BenchmarkArrMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ArrMap(benchInts, func(el int, _ int) int { return el * 2 })
	}
}

func BenchmarkArrFilter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ArrFilter(benchStrings, func(el string, _ int) bool { return el == "keep" })
	}
}

func BenchmarkArrToUnique(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ArrToUnique(benchDups)
	}
}

func BenchmarkArrIncludes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ArrIncludes(benchInts, 999)
	}
}

func BenchmarkArrStrParseInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ArrStrParseInt(benchNumStrs)
	}
}
