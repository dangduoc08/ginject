package utils

import (
	"strconv"
	"strings"
)

func forEach[T any](arr []T, cb func(el T, i int)) {
	for i, el := range arr {
		cb(el, i)
	}
}

func ArrFind[T any](arr []T, cb func(el T, i int) bool) T {
	for i, el := range arr {
		if cb(el, i) {
			return el
		}
	}

	var zero T
	return zero
}

func ArrFindIndex[T any](arr []T, cb func(el T, i int) bool) int {
	for i, el := range arr {
		if cb(el, i) {
			return i
		}
	}

	return -1
}

func ArrMap[T, U any](arr []T, cb func(el T, i int) U) []U {
	newArr := make([]U, len(arr))
	forEach(arr, func(el T, i int) {
		newArr[i] = cb(el, i)
	})

	return newArr
}

func ArrFilter[T any](arr []T, cb func(el T, i int) bool) []T {
	newArr := []T{}
	forEach(arr, func(el T, i int) {
		if cb(el, i) {
			newArr = append(newArr, el)
		}
	})

	return newArr
}

func ArrIncludes[T comparable](arr []T, v T) bool {
	for _, el := range arr {
		if el == v {
			return true
		}
	}

	return false
}

func ArrToUnique[T comparable](arr []T) []T {
	m := make(map[T]bool)
	uniqueArr := []T{}
	for _, el := range arr {
		if !m[el] {
			uniqueArr = append(uniqueArr, el)
			m[el] = true
		}
	}

	return uniqueArr
}

func ArrGet[T any](arr []T, i int) (T, bool) {
	if i >= 0 && i < len(arr) {
		return arr[i], true
	} else {
		var zero T
		return zero, false
	}
}

func ArrParseBool(arr []string) []bool {
	return ArrMap[string, bool](arr, func(el string, i int) bool {
		if boolean, err := strconv.ParseBool(el); err != nil {
			return false
		} else {
			return boolean
		}
	})
}

func ArrParseInt(arr []string) []int {
	return ArrMap[string, int](arr, func(el string, i int) int {
		if intNum, err := strconv.Atoi(el); err != nil {
			return 0
		} else {
			return intNum
		}
	})
}

func ArrParseInt8(arr []string) []int8 {
	return ArrMap[string, int8](arr, func(el string, i int) int8 {
		if i64, err := strconv.ParseInt(el, 10, 8); err != nil {
			return 0
		} else {
			return int8(i64)
		}
	})
}

func ArrParseInt16(arr []string) []int16 {
	return ArrMap[string, int16](arr, func(el string, i int) int16 {
		if i64, err := strconv.ParseInt(el, 10, 16); err != nil {
			return 0
		} else {
			return int16(i64)
		}
	})
}

func ArrParseInt32(arr []string) []int32 {
	return ArrMap[string, int32](arr, func(el string, i int) int32 {
		if i64, err := strconv.ParseInt(el, 10, 32); err != nil {
			return 0
		} else {
			return int32(i64)
		}
	})
}

func ArrParseInt64(arr []string) []int64 {
	return ArrMap[string, int64](arr, func(el string, i int) int64 {
		if i64, err := strconv.ParseInt(el, 10, 64); err != nil {
			return 0
		} else {
			return i64
		}
	})
}

func ArrParseUint(arr []string) []uint {
	return ArrMap[string, uint](arr, func(el string, i int) uint {
		if u64, err := strconv.ParseUint(el, 10, 0); err != nil {
			return 0
		} else {
			return uint(u64)
		}
	})
}

func ArrParseUint8(arr []string) []uint8 {
	return ArrMap[string, uint8](arr, func(el string, i int) uint8 {
		if u64, err := strconv.ParseUint(el, 10, 8); err != nil {
			return 0
		} else {
			return uint8(u64)
		}
	})
}

func ArrParseUint16(arr []string) []uint16 {
	return ArrMap[string, uint16](arr, func(el string, i int) uint16 {
		if u64, err := strconv.ParseUint(el, 10, 16); err != nil {
			return 0
		} else {
			return uint16(u64)
		}
	})
}

func ArrParseUint32(arr []string) []uint32 {
	return ArrMap[string, uint32](arr, func(el string, i int) uint32 {
		if u64, err := strconv.ParseUint(el, 10, 32); err != nil {
			return 0
		} else {
			return uint32(u64)
		}
	})
}

func ArrParseUint64(arr []string) []uint64 {
	return ArrMap[string, uint64](arr, func(el string, i int) uint64 {
		if u64, err := strconv.ParseUint(el, 10, 64); err != nil {
			return 0
		} else {
			return u64
		}
	})
}

func ArrParseFloat32(arr []string) []float32 {
	return ArrMap[string, float32](arr, func(el string, i int) float32 {
		if f64, err := strconv.ParseFloat(el, 32); err != nil {
			return 0
		} else {
			return float32(f64)
		}
	})
}

func ArrParseFloat64(arr []string) []float64 {
	return ArrMap[string, float64](arr, func(el string, i int) float64 {
		if f64, err := strconv.ParseFloat(el, 64); err != nil {
			return 0
		} else {
			return f64
		}
	})
}

func ArrParseComplex64(arr []string) []complex64 {
	return ArrMap[string, complex64](arr, func(el string, i int) complex64 {
		if c128, err := strconv.ParseComplex(strings.ReplaceAll(el, " ", ""), 64); err != nil {
			return 0
		} else {
			return complex64(c128)
		}
	})
}

func ArrParseComplex128(arr []string) []complex128 {
	return ArrMap[string, complex128](arr, func(el string, i int) complex128 {
		if c128, err := strconv.ParseComplex(strings.ReplaceAll(el, " ", ""), 128); err != nil {
			return 0
		} else {
			return c128
		}
	})
}

func ArrParseAny(arr []string) []any {
	return ArrMap[string, any](arr, func(el string, i int) any {
		return el
	})
}
