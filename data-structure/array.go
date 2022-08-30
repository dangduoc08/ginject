package data_structure

func forEach[T any](arr []T, callback func(elem T, index int)) {
	for index, elem := range arr {
		callback(elem, index)
	}
}

func Find[T any](arr []T, callback func(elem T, index int) bool) T {
	for index, elem := range arr {
		if callback(elem, index) {
			return elem
		}
	}

	var nilVal T
	return nilVal
}

func FindIndex[T any](arr []T, callback func(elem T, index int) bool) int {
	for index, elem := range arr {
		if callback(elem, index) {
			return index
		}
	}

	return -1
}

func Map[T, U any](arr []T, callback func(elem T, index int) U) []U {
	newArr := make([]U, 0)
	forEach(arr, func(elem T, index int) {
		newArr = append(newArr, callback(elem, index))
	})

	return newArr
}
