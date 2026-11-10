package slice

import (
	"strings"
	"testing"
)

var benchQueryValues = []string{"12", "345", "6789", "10", "11"}

var escapedResult any

func BenchmarkStrParse(b *testing.B) {
	b.Run("Int_5", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, _ := StrParseInt(benchQueryValues)
			escapedResult = v
		}
	})
	b.Run("Int64_5", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, _ := StrParseInt64(benchQueryValues)
			escapedResult = v
		}
	})
	b.Run("Uint8_5", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, _ := StrParseUint8([]string{"1", "2", "3", "4", "5"})
			escapedResult = v
		}
	})
	b.Run("Float64_5", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, _ := StrParseFloat64(benchQueryValues)
			escapedResult = v
		}
	})
	b.Run("Bool_5", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, _ := StrParseBool([]string{"true", "false", "1", "0", "t"})
			escapedResult = v
		}
	})
	b.Run("Complex128_5", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			v, _ := StrParseComplex128(benchQueryValues)
			escapedResult = v
		}
	})
}

func BenchmarkGet(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Get(benchQueryValues, 2)
	}
}

func BenchmarkMap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Map(benchQueryValues, func(el string, _ int) int { return len(el) })
	}
}

func BenchmarkFilter_BodyKeyPath(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Filter(strings.Split("user.address.city", "."), func(el string, _ int) bool { return strings.TrimSpace(el) != "" })
	}
}

func BenchmarkIter_2D(b *testing.B) {
	arr := []any{[]any{1, 2, 3}, []any{4, 5, 6}}
	b.ReportAllocs()
	for b.Loop() {
		Iter(arr, 2, func(any, int) {})
	}
}
