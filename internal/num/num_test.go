package num

import (
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestF64ToAnyNum(t *testing.T) {
	cases := []struct {
		in   float64
		kind reflect.Kind
		want any
	}{
		{42.9, reflect.Int, int(42)},
		{42.9, reflect.Int8, int8(42)},
		{42.9, reflect.Int16, int16(42)},
		{42.9, reflect.Int32, int32(42)},
		{42.9, reflect.Int64, int64(42)},
		{42.9, reflect.Uint, uint(42)},
		{42.9, reflect.Uint8, uint8(42)},
		{42.9, reflect.Uint16, uint16(42)},
		{42.9, reflect.Uint32, uint32(42)},
		{42.9, reflect.Uint64, uint64(42)},
		{42.5, reflect.Float32, float32(42.5)},
		{42.5, reflect.Float64, float64(42.5)},
		{42.0, reflect.Complex64, complex64(complex(42.0, 0))},
		{42.0, reflect.Complex128, complex(42.0, 0)},
		{-1, reflect.Uint, uint(0)},
		{-1, reflect.Uint8, uint8(0)},
		{-1, reflect.Uint16, uint16(0)},
		{-1, reflect.Uint32, uint32(0)},
		{-1, reflect.Uint64, uint64(0)},
		{1, reflect.String, 0},
	}

	for _, c := range cases {
		got := F64ToAnyNum(c.in, c.kind)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "NumF64ToAnyNum"))
		}
	}
}
