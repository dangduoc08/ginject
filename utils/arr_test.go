package utils

import (
	"strings"
	"testing"
)

func TestArrFind(t *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}

	expect1 := "Jane Doe"
	output1 := ArrFind(arr, func(el string, i int) bool {
		return el == expect1
	})

	if output1 != expect1 {
		t.Errorf("ArrFind = %v; expect = %v", output1, expect1)
	}
}

func TestArrFindIndex(t *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}

	expect1 := 1
	output1 := ArrFindIndex(arr, func(el string, i int) bool {
		return el == "Jane Doe"
	})

	if output1 != expect1 {
		t.Errorf("ArrFindIndex = %v; expect = %v", output1, expect1)
	}
}

func TestArrMap(t *testing.T) {
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
	output1 := ArrMap(arr, func(el string, i int) int {
		i++
		return i
	})

	if output1[0] != expect1[0] ||
		output1[1] != expect1[1] ||
		output1[2] != expect1[2] {
		t.Errorf("ArrMap = %v; expect = %v", output1, expect1)
	}
}

func TestArrFilter(t *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}

	expect1 := []string{
		"John Doe",
		"Jane Doe",
	}
	output1 := ArrFilter(arr, func(el string, i int) bool {
		return strings.Contains(el, "Doe")
	})

	if output1[0] != expect1[0] ||
		output1[1] != expect1[1] ||
		len(output1) > 2 {
		t.Errorf("ArrFilter = %v; expect = %v", output1, expect1)
	}
}

func TestArrIncludes(t *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}

	expect1 := true
	output1 := ArrIncludes(arr, arr[0])

	if output1 != expect1 {
		t.Errorf("ArrIncludes = %v; expect = %v", output1, expect1)
	}

	expect2 := false
	output2 := ArrIncludes(arr, arr[1]+"Foz")

	if output2 != expect2 {
		t.Errorf("ArrIncludes = %v; expect = %v", output2, expect2)
	}
}

func TestArrToUnique(t *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
		"John Doe",
		"Jane Doe",
	}

	expect1 := []string{
		"John Doe",
		"Jane Doe",
		"The Rock",
	}
	output1 := ArrToUnique(arr)

	if len(expect1) != len(output1) {
		t.Errorf("len(expect1) = %v; len(output1) = %v", len(expect1), len(output1))
	}

	for i, e := range output1 {
		if expect1[i] != e {
			t.Errorf("expect1's element at index %v = %v; output1's element at index %v = %v", i, expect1[i], i, e)
		}
	}
}
