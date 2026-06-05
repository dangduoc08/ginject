package slice

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestFind(t *testing.T) {
	arr := []string{"John Doe", "Jane Doe", "The Rock"}
	expect1 := "Jane Doe"
	output1 := Find(arr, func(el string, i int) bool { return el == expect1 })
	if output1 != expect1 {
		t.Error(test.DiffMessage(output1, expect1, "ArrFind"))
	}
}

func TestFindIndex(t *testing.T) {
	arr := []string{"John Doe", "Jane Doe", "The Rock"}
	expect1 := 1
	output1 := FindIndex(arr, func(el string, i int) bool { return el == "Jane Doe" })
	if output1 != expect1 {
		t.Error(test.DiffMessage(output1, expect1, "ArrFindIndex"))
	}
}

func TestMap(t *testing.T) {
	arr := []string{"John Doe", "Jane Doe", "The Rock"}
	expect1 := []int{1, 2, 3}
	output1 := Map(arr, func(el string, i int) int { return i + 1 })
	if output1[0] != expect1[0] || output1[1] != expect1[1] || output1[2] != expect1[2] {
		t.Error(test.DiffMessage(output1, expect1, "ArrMap"))
	}
}

func TestFilter(t *testing.T) {
	arr := []string{"John Doe", "Jane Doe", "The Rock"}
	expect1 := []string{"John Doe", "Jane Doe"}
	output1 := Filter(arr, func(el string, i int) bool { return strings.Contains(el, "Doe") })
	if output1[0] != expect1[0] || output1[1] != expect1[1] || len(output1) > 2 {
		t.Error(test.DiffMessage(output1, expect1, "ArrFilter"))
	}
}

func TestToUnique(t *testing.T) {
	arr := []string{"John Doe", "Jane Doe", "The Rock", "John Doe", "Jane Doe"}
	expect1 := []string{"John Doe", "Jane Doe", "The Rock"}
	output1 := ToUnique(arr)
	if len(expect1) != len(output1) {
		t.Error(test.DiffMessage(len(output1), len(expect1), "ArrToUnique length"))
	}
	for i, e := range output1 {
		if expect1[i] != e {
			t.Error(test.DiffMessage(e, expect1[i], fmt.Sprintf("ArrToUnique element at index %d", i)))
		}
	}
}

func TestGet(t *testing.T) {
	arr := []string{"John Doe", "Jane Doe"}
	output1, _ := Get(arr, 0)
	if output1 != "John Doe" {
		t.Error(test.DiffMessage(output1, "John Doe", "ArrGet valid index"))
	}
	output2, _ := Get(arr, 100)
	if output2 != "" {
		t.Error(test.DiffMessage(output2, "", "ArrGet out-of-bounds"))
	}
}

func TestArrFindNotFound(t *testing.T) {
	arr := []int{1, 2, 3}
	got := Find(arr, func(el int, _ int) bool { return el == 99 })
	if got != 0 {
		t.Error(test.DiffMessage(got, 0, "ArrFind not-found should return zero value"))
	}
}

func TestArrFindIndexNotFound(t *testing.T) {
	arr := []int{1, 2, 3}
	got := FindIndex(arr, func(el int, _ int) bool { return el == 99 })
	if got != -1 {
		t.Error(test.DiffMessage(got, -1, "ArrFindIndex not-found should return -1"))
	}
}

func TestArrGetBool(t *testing.T) {
	arr := []string{"a"}
	if _, ok := Get(arr, 0); !ok {
		t.Error(test.DiffMessage(false, true, "ArrGet valid index should return ok=true"))
	}
	if _, ok := Get(arr, -1); ok {
		t.Error(test.DiffMessage(true, false, "ArrGet negative index should return ok=false"))
	}
	if _, ok := Get(arr, 1); ok {
		t.Error(test.DiffMessage(true, false, "ArrGet out-of-bounds index should return ok=false"))
	}
}

func TestStrParseBool(t *testing.T) {
	got := StrParseBool([]string{"true", "false", "bad"})
	want := []bool{true, false, false}
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Error(test.DiffMessage(got, want, "ArrStrParseBool"))
	}
}

func TestStrParseInt(t *testing.T) {
	got := StrParseInt([]string{"1", "-2", "bad"})
	want := []int{1, -2, 0}
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Error(test.DiffMessage(got, want, "ArrStrParseInt"))
	}
}

func TestStrParseFloat64(t *testing.T) {
	got := StrParseFloat64([]string{"1.5", "-2.5", "bad"})
	want := []float64{1.5, -2.5, 0}
	if got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Error(test.DiffMessage(got, want, "ArrStrParseFloat64"))
	}
}

func TestStrParseComplex64(t *testing.T) {
	got := StrParseComplex64([]string{"1+2i", "bad"})
	want := []complex64{complex(1, 2), 0}
	if got[0] != want[0] || got[1] != want[1] {
		t.Error(test.DiffMessage(got, want, "ArrStrParseComplex64"))
	}
}

func TestStrParseAny(t *testing.T) {
	got := StrParseAny([]string{"a", "b"})
	if got[0] != "a" || got[1] != "b" {
		t.Error(test.DiffMessage(got, []any{"a", "b"}, "ArrStrParseAny"))
	}
}

func TestArrIterMultiDimensions(t *testing.T) {
	multiDimension := []any{
		[]any{
			[]any{
				[]any{1000, 2000, 3000, 4000},
				[]any{4001, 5001, 6001, 7001},
			},
			[]any{
				[]any{1010, 2010, 3010, 4010},
				[]any{4011, 5011, 6011, 7011},
			},
		},
		[]any{
			[]any{
				[]any{1100, 2100, 3100, 4100},
				[]any{4101, 5101, 6101, 7101},
			},
			[]any{
				[]any{1110, 2110, 3110, 4110},
				[]any{4111, 5111, 6111, 7111},
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
	Iter(multiDimension, 4, func(el any, d int) {
		if reflect.TypeOf(el).Kind() == reflect.Int {
			result1 += el.(int)
		}
	})

	if expected1 != result1 {
		t.Error(test.DiffMessage(result1, expected1, "ArrIter sum"))
	}
}
