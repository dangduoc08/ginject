package ds

import (
	"testing"
)

func TestFind(test *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}

	expect1 := "Jane Doe"
	output1 := Find(arr, func(elem string, index int) bool {
		return elem == expect1
	})

	if output1 != expect1 {
		test.Errorf("Find = %v; expect = %v", output1, expect1)
	}
}

func TestFindIndex(test *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}

	expect1 := 1
	output1 := FindIndex(arr, func(elem string, index int) bool {
		return elem == "Jane Doe"
	})

	if output1 != expect1 {
		test.Errorf("FindIndex = %v; expect = %v", output1, expect1)
	}
}

func TestMap(test *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}

	expect1 := []int{
		1,
		2,
		3,
	}
	output1 := Map(arr, func(elem string, index int) int {
		index++
		return index
	})

	if output1[0] != expect1[0] ||
		output1[1] != expect1[1] ||
		output1[2] != expect1[2] {
		test.Errorf("Map = %v; expect = %v", output1, expect1)
	}
}
