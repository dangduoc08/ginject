package slice

import (
	"strconv"
	"strings"
)

func Find[T any](arr []T, cb func(el T, i int) bool) T {
	for i, el := range arr {
		if cb(el, i) {
			return el
		}
	}

	var zero T
	return zero
}

func FindIndex[T any](arr []T, cb func(el T, i int) bool) int {
	for i, el := range arr {
		if cb(el, i) {
			return i
		}
	}

	return -1
}

func Map[T, U any](arr []T, cb func(el T, i int) U) []U {
	newArr := make([]U, len(arr))
	for i, el := range arr {
		newArr[i] = cb(el, i)
	}
	return newArr
}

func Filter[T any](arr []T, cb func(el T, i int) bool) []T {
	newArr := make([]T, 0, len(arr))
	for i, el := range arr {
		if cb(el, i) {
			newArr = append(newArr, el)
		}
	}
	return newArr
}

func ToUnique[T comparable](arr []T) []T {
	m := make(map[T]struct{}, len(arr))
	uniqueArr := make([]T, 0, len(arr))
	for _, el := range arr {
		if _, seen := m[el]; !seen {
			uniqueArr = append(uniqueArr, el)
			m[el] = struct{}{}
		}
	}
	return uniqueArr
}

func Get[T any](arr []T, i int) (T, bool) {
	if i >= 0 && i < len(arr) {
		return arr[i], true
	}
	var zero T
	return zero, false
}

func StrParseBool(arr []string) []bool {
	return Map(arr, func(el string, i int) bool {
		if boolean, err := strconv.ParseBool(el); err != nil {
			return false
		} else {
			return boolean
		}
	})
}

func StrParseInt(arr []string) []int {
	return Map(arr, func(el string, i int) int {
		if intNum, err := strconv.Atoi(el); err != nil {
			return 0
		} else {
			return intNum
		}
	})
}

func StrParseInt8(arr []string) []int8 {
	return Map(arr, func(el string, i int) int8 {
		if i64, err := strconv.ParseInt(el, 10, 8); err != nil {
			return 0
		} else {
			return int8(i64)
		}
	})
}

func StrParseInt16(arr []string) []int16 {
	return Map(arr, func(el string, i int) int16 {
		if i64, err := strconv.ParseInt(el, 10, 16); err != nil {
			return 0
		} else {
			return int16(i64)
		}
	})
}

func StrParseInt32(arr []string) []int32 {
	return Map(arr, func(el string, i int) int32 {
		if i64, err := strconv.ParseInt(el, 10, 32); err != nil {
			return 0
		} else {
			return int32(i64)
		}
	})
}

func StrParseInt64(arr []string) []int64 {
	return Map(arr, func(el string, i int) int64 {
		if i64, err := strconv.ParseInt(el, 10, 64); err != nil {
			return 0
		} else {
			return i64
		}
	})
}

func StrParseUint(arr []string) []uint {
	return Map(arr, func(el string, i int) uint {
		if u64, err := strconv.ParseUint(el, 10, 0); err != nil {
			return 0
		} else {
			return uint(u64)
		}
	})
}

func StrParseUint8(arr []string) []uint8 {
	return Map(arr, func(el string, i int) uint8 {
		if u64, err := strconv.ParseUint(el, 10, 8); err != nil {
			return 0
		} else {
			return uint8(u64)
		}
	})
}

func StrParseUint16(arr []string) []uint16 {
	return Map(arr, func(el string, i int) uint16 {
		if u64, err := strconv.ParseUint(el, 10, 16); err != nil {
			return 0
		} else {
			return uint16(u64)
		}
	})
}

func StrParseUint32(arr []string) []uint32 {
	return Map(arr, func(el string, i int) uint32 {
		if u64, err := strconv.ParseUint(el, 10, 32); err != nil {
			return 0
		} else {
			return uint32(u64)
		}
	})
}

func StrParseUint64(arr []string) []uint64 {
	return Map(arr, func(el string, i int) uint64 {
		if u64, err := strconv.ParseUint(el, 10, 64); err != nil {
			return 0
		} else {
			return u64
		}
	})
}

func StrParseFloat32(arr []string) []float32 {
	return Map(arr, func(el string, i int) float32 {
		if f64, err := strconv.ParseFloat(el, 32); err != nil {
			return 0
		} else {
			return float32(f64)
		}
	})
}

func StrParseFloat64(arr []string) []float64 {
	return Map(arr, func(el string, i int) float64 {
		if f64, err := strconv.ParseFloat(el, 64); err != nil {
			return 0
		} else {
			return f64
		}
	})
}

func StrParseComplex64(arr []string) []complex64 {
	return Map(arr, func(el string, i int) complex64 {
		if c128, err := strconv.ParseComplex(strings.ReplaceAll(el, " ", ""), 64); err != nil {
			return 0
		} else {
			return complex64(c128)
		}
	})
}

func StrParseComplex128(arr []string) []complex128 {
	return Map(arr, func(el string, i int) complex128 {
		if c128, err := strconv.ParseComplex(strings.ReplaceAll(el, " ", ""), 128); err != nil {
			return 0
		} else {
			return c128
		}
	})
}

func StrParseAny(arr []string) []any {
	return Map(arr, func(el string, i int) any {
		return el
	})
}

func Iter(arr []any, dimmensions int, cb func(any, int)) {
	for _, el := range arr {
		if sub, ok := el.([]any); ok {
			Iter(sub, dimmensions-1, cb)
		}
		cb(el, dimmensions)
	}
}
