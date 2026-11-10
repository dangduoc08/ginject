package num

import (
	"math"
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestF64ToAnyNum(t *testing.T) {
	cases := []struct {
		name   string
		in     float64
		kind   reflect.Kind
		want   any
		wantOK bool
	}{
		{"int", 42, reflect.Int, int(42), true},
		{"int8", 42, reflect.Int8, int8(42), true},
		{"int16", 42, reflect.Int16, int16(42), true},
		{"int32", 42, reflect.Int32, int32(42), true},
		{"int64", 42, reflect.Int64, int64(42), true},
		{"uint", 42, reflect.Uint, uint(42), true},
		{"uint8", 42, reflect.Uint8, uint8(42), true},
		{"uint16", 42, reflect.Uint16, uint16(42), true},
		{"uint32", 42, reflect.Uint32, uint32(42), true},
		{"uint64", 42, reflect.Uint64, uint64(42), true},
		{"float32", 42.5, reflect.Float32, float32(42.5), true},
		{"float64", 42.5, reflect.Float64, 42.5, true},
		{"complex64", 42, reflect.Complex64, complex64(complex(42, 0)), true},
		{"complex128", 42, reflect.Complex128, complex(42, 0), true},

		{"int8 max", 127, reflect.Int8, int8(127), true},
		{"int8 min", -128, reflect.Int8, int8(-128), true},
		{"int8 over", 128, reflect.Int8, nil, false},
		{"int8 under", -129, reflect.Int8, nil, false},
		{"int8 300 does not wrap", 300, reflect.Int8, nil, false},
		{"int16 over", 32768, reflect.Int16, nil, false},
		{"int32 over", 2147483648, reflect.Int32, nil, false},
		{"int32 min", -2147483648, reflect.Int32, int32(math.MinInt32), true},
		{"int64 min", -9223372036854775808, reflect.Int64, int64(math.MinInt64), true},
		{"int64 2^63 over", 9223372036854775808, reflect.Int64, nil, false},
		{"int64 1e20 does not wrap", 1e20, reflect.Int64, nil, false},
		{"uint8 max", 255, reflect.Uint8, uint8(255), true},
		{"uint8 over", 256, reflect.Uint8, nil, false},
		{"uint8 300 does not wrap", 300, reflect.Uint8, nil, false},
		{"uint16 over", 65536, reflect.Uint16, nil, false},
		{"uint32 over", 4294967296, reflect.Uint32, nil, false},
		{"uint64 2^64 over", 18446744073709551616, reflect.Uint64, nil, false},
		{"uint negative", -1, reflect.Uint, nil, false},
		{"uint8 negative", -1, reflect.Uint8, nil, false},
		{"uint negative zero", math.Copysign(0, -1), reflect.Uint, uint(0), true},

		{"int fraction", 1.9, reflect.Int, nil, false},
		{"int negative fraction", -1.9, reflect.Int, nil, false},
		{"int whole float", 1.0, reflect.Int, int(1), true},
		{"uint fraction", 0.5, reflect.Uint, nil, false},

		{"float32 max", math.MaxFloat32, reflect.Float32, float32(math.MaxFloat32), true},
		{"float32 over", 3.4e39, reflect.Float32, nil, false},
		{"complex64 over", 3.4e39, reflect.Complex64, nil, false},

		{"NaN", math.NaN(), reflect.Float64, nil, false},
		{"+Inf", math.Inf(1), reflect.Float64, nil, false},
		{"-Inf", math.Inf(-1), reflect.Int, nil, false},
		{"float32 NaN", math.NaN(), reflect.Float32, nil, false},
		{"complex64 NaN", math.NaN(), reflect.Complex64, nil, false},
		{"complex128 NaN", math.NaN(), reflect.Complex128, nil, false},
		{"int NaN", math.NaN(), reflect.Int, nil, false},
		{"uint64 +Inf", math.Inf(1), reflect.Uint64, nil, false},
		{"uint64 max float below 2^64", 18446744073709549568, reflect.Uint64, uint64(18446744073709549568), true},
		{"unsupported kind", 1, reflect.String, nil, false},
	}

	for _, c := range cases {
		got, ok := F64ToAnyNum(c.in, c.kind)
		if ok != c.wantOK || got != c.want {
			t.Error(test.DiffMessage([]any{got, ok}, []any{c.want, c.wantOK}, c.name))
		}
	}
}
