package num

import (
	"math"
	"math/bits"
	"reflect"
)

type integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

const (
	intLimit  = 1 << (bits.UintSize - 1)
	uintLimit = 1 << bits.UintSize
)

func F64ToAnyNum(f64 float64, kind reflect.Kind) (any, bool) {
	switch kind {
	case reflect.Int:
		return toInteger[int](f64, -intLimit, intLimit)
	case reflect.Int8:
		return toInteger[int8](f64, math.MinInt8, 1<<7)
	case reflect.Int16:
		return toInteger[int16](f64, math.MinInt16, 1<<15)
	case reflect.Int32:
		return toInteger[int32](f64, math.MinInt32, 1<<31)
	case reflect.Int64:
		return toInteger[int64](f64, math.MinInt64, 1<<63)
	case reflect.Uint:
		return toInteger[uint](f64, 0, uintLimit)
	case reflect.Uint8:
		return toInteger[uint8](f64, 0, 1<<8)
	case reflect.Uint16:
		return toInteger[uint16](f64, 0, 1<<16)
	case reflect.Uint32:
		return toInteger[uint32](f64, 0, 1<<32)
	case reflect.Uint64:
		return toInteger[uint64](f64, 0, 1<<64)
	case reflect.Float32:
		if !isFinite(f64) || math.Abs(f64) > math.MaxFloat32 {
			return nil, false
		}
		return float32(f64), true
	case reflect.Float64:
		if !isFinite(f64) {
			return nil, false
		}
		return f64, true
	case reflect.Complex64:
		if !isFinite(f64) || math.Abs(f64) > math.MaxFloat32 {
			return nil, false
		}
		return complex64(complex(f64, 0)), true
	case reflect.Complex128:
		if !isFinite(f64) {
			return nil, false
		}
		return complex(f64, 0), true
	}

	return nil, false
}

func toInteger[T integer](f64, low, high float64) (any, bool) {
	if !(f64 >= low && f64 < high) {
		return nil, false
	}
	v := T(f64)
	if float64(v) != f64 {
		return nil, false
	}
	return v, true
}

func isFinite(f64 float64) bool {
	return !math.IsNaN(f64) && !math.IsInf(f64, 0)
}
