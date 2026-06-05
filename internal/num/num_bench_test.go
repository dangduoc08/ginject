package num

import (
	"reflect"
	"testing"
)

func BenchmarkF64ToAnyNum_Int(b *testing.B) {
	for i := 0; i < b.N; i++ {
		F64ToAnyNum(42.9, reflect.Int)
	}
}

func BenchmarkF64ToAnyNum_Uint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		F64ToAnyNum(42.9, reflect.Uint)
	}
}

func BenchmarkF64ToAnyNum_UintNeg(b *testing.B) {
	for i := 0; i < b.N; i++ {
		F64ToAnyNum(-1, reflect.Uint)
	}
}

func BenchmarkF64ToAnyNum_Float64(b *testing.B) {
	for i := 0; i < b.N; i++ {
		F64ToAnyNum(42.5, reflect.Float64)
	}
}
