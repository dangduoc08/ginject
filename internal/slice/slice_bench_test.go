package slice

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

func BenchmarkMap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Map(benchInts, func(el int, _ int) int { return el * 2 })
	}
}

func BenchmarkFilter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Filter(benchStrings, func(el string, _ int) bool { return el == "keep" })
	}
}

func BenchmarkToUnique(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ToUnique(benchDups)
	}
}

func BenchmarkStrParseInt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		StrParseInt(benchNumStrs)
	}
}
