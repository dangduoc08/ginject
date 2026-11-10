package slice

import (
	"strconv"
	"strings"
)

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

func Get[T any](arr []T, i int) (T, bool) {
	if i >= 0 && i < len(arr) {
		return arr[i], true
	}
	var zero T
	return zero, false
}

func StrParseBool(arr []string) ([]bool, bool) {
	return parseAll(arr, strconv.ParseBool)
}

func StrParseInt(arr []string) ([]int, bool) {
	return parseAll(arr, strconv.Atoi)
}

func StrParseInt8(arr []string) ([]int8, bool) {
	return parseAll(arr, func(s string) (int8, error) {
		v, err := strconv.ParseInt(s, 10, 8)
		return int8(v), err
	})
}

func StrParseInt16(arr []string) ([]int16, bool) {
	return parseAll(arr, func(s string) (int16, error) {
		v, err := strconv.ParseInt(s, 10, 16)
		return int16(v), err
	})
}

func StrParseInt32(arr []string) ([]int32, bool) {
	return parseAll(arr, func(s string) (int32, error) {
		v, err := strconv.ParseInt(s, 10, 32)
		return int32(v), err
	})
}

func StrParseInt64(arr []string) ([]int64, bool) {
	return parseAll(arr, func(s string) (int64, error) {
		return strconv.ParseInt(s, 10, 64)
	})
}

func StrParseUint(arr []string) ([]uint, bool) {
	return parseAll(arr, func(s string) (uint, error) {
		v, err := strconv.ParseUint(s, 10, 0)
		return uint(v), err
	})
}

func StrParseUint8(arr []string) ([]uint8, bool) {
	return parseAll(arr, func(s string) (uint8, error) {
		v, err := strconv.ParseUint(s, 10, 8)
		return uint8(v), err
	})
}

func StrParseUint16(arr []string) ([]uint16, bool) {
	return parseAll(arr, func(s string) (uint16, error) {
		v, err := strconv.ParseUint(s, 10, 16)
		return uint16(v), err
	})
}

func StrParseUint32(arr []string) ([]uint32, bool) {
	return parseAll(arr, func(s string) (uint32, error) {
		v, err := strconv.ParseUint(s, 10, 32)
		return uint32(v), err
	})
}

func StrParseUint64(arr []string) ([]uint64, bool) {
	return parseAll(arr, func(s string) (uint64, error) {
		return strconv.ParseUint(s, 10, 64)
	})
}

func StrParseFloat32(arr []string) ([]float32, bool) {
	return parseAll(arr, func(s string) (float32, error) {
		v, err := strconv.ParseFloat(s, 32)
		return float32(v), err
	})
}

func StrParseFloat64(arr []string) ([]float64, bool) {
	return parseAll(arr, func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	})
}

func StrParseComplex64(arr []string) ([]complex64, bool) {
	return parseAll(arr, func(s string) (complex64, error) {
		v, err := strconv.ParseComplex(strings.ReplaceAll(s, " ", ""), 64)
		return complex64(v), err
	})
}

func StrParseComplex128(arr []string) ([]complex128, bool) {
	return parseAll(arr, func(s string) (complex128, error) {
		return strconv.ParseComplex(strings.ReplaceAll(s, " ", ""), 128)
	})
}

func StrParseAny(arr []string) []any {
	return Map(arr, func(el string, i int) any {
		return el
	})
}

func parseAll[T any](arr []string, parse func(string) (T, error)) ([]T, bool) {
	out := make([]T, len(arr))
	for i, el := range arr {
		v, err := parse(el)
		if err != nil {
			return nil, false
		}
		out[i] = v
	}
	return out, true
}

func Iter(arr []any, dimensions int, cb func(any, int)) {
	for _, el := range arr {
		if sub, ok := el.([]any); ok {
			Iter(sub, dimensions-1, cb)
		}
		cb(el, dimensions)
	}
}
