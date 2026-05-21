package utils

import (
	"reflect"
	"testing"
)

func BenchmarkNumF64ToAnyNum_Int(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NumF64ToAnyNum(42.9, reflect.Int)
	}
}

func BenchmarkNumF64ToAnyNum_Uint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NumF64ToAnyNum(42.9, reflect.Uint)
	}
}

func BenchmarkNumF64ToAnyNum_UintNeg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NumF64ToAnyNum(-1, reflect.Uint)
	}
}

func BenchmarkNumF64ToAnyNum_Float64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NumF64ToAnyNum(42.5, reflect.Float64)
	}
}
