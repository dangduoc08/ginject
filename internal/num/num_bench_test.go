package num

import (
	"reflect"
	"testing"
)

func BenchmarkF64ToAnyNum(b *testing.B) {
	cases := []struct {
		name string
		in   float64
		kind reflect.Kind
	}{
		{"int_small", 42, reflect.Int},
		{"int_large", 12345, reflect.Int},
		{"int64_id", 1700000000123, reflect.Int64},
		{"uint8", 200, reflect.Uint8},
		{"float64", 3.14, reflect.Float64},
		{"int_overflow", 1e20, reflect.Int64},
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				F64ToAnyNum(c.in, c.kind)
			}
		})
	}
}
