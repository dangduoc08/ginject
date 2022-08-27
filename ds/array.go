package ds

func ForEach[T any](arr []T, cb func(elem T, index int, arr []T)) {
	for i, v := range arr {
		cb(v, i, arr)
	}
}

func Find[T any](arr []T, cb func(elem T, index int, arr []T) bool) T {
	for i, v := range arr {
		if cb(v, i, arr) {
			return v
		}
	}
	var v T
	return v
}

func Map[T, U any](arr []T, cb func(elem T, index int, arr []T) U) []U {
	newArr := make([]U, 0)
	ForEach(arr, func(elem T, index int, arr []T) {
		newArr = append(newArr, cb(elem, index, arr))
	})
	return newArr
}
