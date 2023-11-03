package utils

import (
	"reflect"
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

func TestArrGet(t *testing.T) {
	arr := []string{
		"John Doe",
		"Jane Doe",
	}

	expect1 := "John Doe"
	output1, _ := ArrGet[string](arr, 0)
	if expect1 != output1 {
		t.Errorf("len(expect1) = %v; len(output1) = %v", len(expect1), len(output1))
	}

	expect2 := ""
	output2, _ := ArrGet[string](arr, 100)
	if expect2 != output2 {
		t.Errorf("len(expect1) = %v; len(output1) = %v", len(expect1), len(output1))
	}
}

func TestArrIterMultiDimensions(t *testing.T) {
	multiDimension := []any{
		[]any{
			[]any{
				[]any{
					1000, 2000, 3000, 4000,
				},
				[]any{
					4001, 5001, 6001, 7001,
				},
			},
			[]any{
				[]any{
					1010, 2010, 3010, 4010,
				},
				[]any{
					4011, 5011, 6011, 7011,
				},
			},
		},
		[]any{
			[]any{
				[]any{
					1100, 2100, 3100, 4100,
				},
				[]any{
					4101, 5101, 6101, 7101,
				},
			},
			[]any{
				[]any{
					1110, 2110, 3110, 4110,
				},
				[]any{
					4111, 5111, 6111, 7111,
				},
			},
		},
	}

	expected1 := 1000 + 2000 + 3000 + 4000 +
		4001 + 5001 + 6001 + 7001 +
		1010 + 2010 + 3010 + 4010 +
		4011 + 5011 + 6011 + 7011 +
		1100 + 2100 + 3100 + 4100 +
		4101 + 5101 + 6101 + 7101 +
		1110 + 2110 + 3110 + 4110 +
		4111 + 5111 + 6111 + 7111

	result1 := 0
	ArrIter(multiDimension, 4, func(el any, d int) {
		if reflect.TypeOf(el).Kind() == reflect.Int {
			result1 += el.(int)
		}
	})

	if expected1 != result1 {
		t.Errorf("expected1 = %v; result1 = %v", expected1, result1)
	}
}
