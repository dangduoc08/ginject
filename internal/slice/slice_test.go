package slice

import (
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestMap(t *testing.T) {
	got := Map([]string{"a", "bb", "ccc"}, func(el string, i int) int { return len(el) + i })
	want := []int{1, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Error(test.DiffMessage(got, want, "Map"))
	}
	if got := Map([]int(nil), func(el int, _ int) int { return el }); got == nil || len(got) != 0 {
		t.Error(test.DiffMessage(got, []int{}, "Map of nil returns an empty, non-nil slice"))
	}
}

func TestFilter(t *testing.T) {
	got := Filter([]string{"user", "", " ", "city"}, func(el string, _ int) bool { return el != "" && el != " " })
	want := []string{"user", "city"}
	if !reflect.DeepEqual(got, want) {
		t.Error(test.DiffMessage(got, want, "Filter"))
	}
	if got := Filter([]int{1, 2}, func(int, int) bool { return false }); got == nil || len(got) != 0 {
		t.Error(test.DiffMessage(got, []int{}, "Filter keeping nothing returns an empty, non-nil slice"))
	}
}

func TestGet(t *testing.T) {
	arr := []string{"a", "b"}
	cases := []struct {
		name   string
		i      int
		want   string
		wantOK bool
	}{
		{"first", 0, "a", true},
		{"last", 1, "b", true},
		{"past end", 2, "", false},
		{"negative", -1, "", false},
	}
	for _, c := range cases {
		got, ok := Get(arr, c.i)
		if got != c.want || ok != c.wantOK {
			t.Error(test.DiffMessage([]any{got, ok}, []any{c.want, c.wantOK}, "Get "+c.name))
		}
	}
	if got, ok := Get([]int(nil), 0); got != 0 || ok {
		t.Error(test.DiffMessage([]any{got, ok}, []any{0, false}, "Get on nil"))
	}
}

func TestStrParse(t *testing.T) {
	cases := []struct {
		name   string
		parse  func() (any, bool)
		want   any
		wantOK bool
	}{
		{"Bool", wrap(StrParseBool([]string{"true", "0", "T"})), []bool{true, false, true}, true},
		{"Bool yes", wrap(StrParseBool([]string{"yes"})), nil, false},
		{"Int", wrap(StrParseInt([]string{"1", "-2", "3"})), []int{1, -2, 3}, true},
		{"Int abc", wrap(StrParseInt([]string{"1", "abc", "3"})), nil, false},
		{"Int leading space", wrap(StrParseInt([]string{" 7"})), nil, false},
		{"Int8 max", wrap(StrParseInt8([]string{"127", "-128"})), []int8{127, -128}, true},
		{"Int8 overflow", wrap(StrParseInt8([]string{"300"})), nil, false},
		{"Int16", wrap(StrParseInt16([]string{"32767"})), []int16{32767}, true},
		{"Int16 overflow", wrap(StrParseInt16([]string{"32768"})), nil, false},
		{"Int32", wrap(StrParseInt32([]string{"-2147483648"})), []int32{-2147483648}, true},
		{"Int32 overflow", wrap(StrParseInt32([]string{"2147483648"})), nil, false},
		{"Int64", wrap(StrParseInt64([]string{"9007199254740993"})), []int64{9007199254740993}, true},
		{"Int64 overflow", wrap(StrParseInt64([]string{"9223372036854775808"})), nil, false},
		{"Uint", wrap(StrParseUint([]string{"42"})), []uint{42}, true},
		{"Uint negative", wrap(StrParseUint([]string{"-1"})), nil, false},
		{"Uint8", wrap(StrParseUint8([]string{"255"})), []uint8{255}, true},
		{"Uint8 overflow", wrap(StrParseUint8([]string{"256"})), nil, false},
		{"Uint16", wrap(StrParseUint16([]string{"65535"})), []uint16{65535}, true},
		{"Uint16 overflow", wrap(StrParseUint16([]string{"65536"})), nil, false},
		{"Uint32", wrap(StrParseUint32([]string{"4294967295"})), []uint32{4294967295}, true},
		{"Uint32 overflow", wrap(StrParseUint32([]string{"4294967296"})), nil, false},
		{"Uint64", wrap(StrParseUint64([]string{"18446744073709551615"})), []uint64{18446744073709551615}, true},
		{"Uint64 overflow", wrap(StrParseUint64([]string{"18446744073709551616"})), nil, false},
		{"Float32", wrap(StrParseFloat32([]string{"1.5"})), []float32{1.5}, true},
		{"Float32 overflow", wrap(StrParseFloat32([]string{"1e39"})), nil, false},
		{"Float64", wrap(StrParseFloat64([]string{"3.14", "-2"})), []float64{3.14, -2}, true},
		{"Float64 overflow", wrap(StrParseFloat64([]string{"1e400"})), nil, false},
		{"Float64 garbage", wrap(StrParseFloat64([]string{"pi"})), nil, false},
		{"Complex64", wrap(StrParseComplex64([]string{"1 + 2i"})), []complex64{complex(1, 2)}, true},
		{"Complex64 garbage", wrap(StrParseComplex64([]string{"i+"})), nil, false},
		{"Complex128", wrap(StrParseComplex128([]string{"3-4i"})), []complex128{complex(3, -4)}, true},
		{"Complex128 garbage", wrap(StrParseComplex128([]string{"x"})), nil, false},
		{"Int empty input", wrap(StrParseInt([]string{})), []int{}, true},
	}

	for _, c := range cases {
		got, ok := c.parse()
		if ok != c.wantOK || (c.wantOK && !reflect.DeepEqual(got, c.want)) || (!c.wantOK && !reflect.ValueOf(got).IsNil()) {
			t.Error(test.DiffMessage([]any{got, ok}, []any{c.want, c.wantOK}, "StrParse"+c.name))
		}
	}
}

func wrap[T any](v []T, ok bool) func() (any, bool) {
	return func() (any, bool) { return v, ok }
}

func TestStrParseAny(t *testing.T) {
	got := StrParseAny([]string{"a", "1"})
	want := []any{"a", "1"}
	if !reflect.DeepEqual(got, want) {
		t.Error(test.DiffMessage(got, want, "StrParseAny"))
	}
}

func TestIter(t *testing.T) {
	arr := []any{1, []any{2, []any{3}}, 4}
	type visit struct {
		el  any
		dim int
	}
	var got []visit
	Iter(arr, 3, func(el any, dim int) { got = append(got, visit{el, dim}) })

	want := []visit{
		{1, 3},
		{2, 2},
		{3, 1},
		{[]any{3}, 2},
		{[]any{2, []any{3}}, 3},
		{4, 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Error(test.DiffMessage(got, want, "Iter visits leaves and sub-arrays, deepest first"))
	}
}
