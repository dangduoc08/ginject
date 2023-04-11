package utils

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
