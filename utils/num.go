package utils

import (
	"reflect"
)

func NumF64ToAnyNum(f64 float64, t reflect.Kind) any {
	switch t {
	case reflect.Int:
		return int(f64)

	case reflect.Int8:
		return int8(f64)

	case reflect.Int16:
		return int16(f64)

	case reflect.Int32:
		return int32(f64)

	case reflect.Int64:
		return int64(f64)

	case reflect.Uint:
		if f64 < 0 {
			return 0
		}
		return uint(f64)

	case reflect.Uint8:
		if f64 < 0 {
			return 0
		}
		return uint8(f64)

	case reflect.Uint16:
		if f64 < 0 {
			return 0
		}
		return uint16(f64)

	case reflect.Uint32:
		if f64 < 0 {
			return 0
		}
		return uint32(f64)

	case reflect.Uint64:
		if f64 < 0 {
			return 0
		}
		return uint64(f64)

	case reflect.Float32:
		return float32(f64)

	case reflect.Float64:
		return f64

	case reflect.Complex64:
		return complex64(complex(f64, 0))

	case reflect.Complex128:
		return complex(f64, 0)
	}

	return 0
}
