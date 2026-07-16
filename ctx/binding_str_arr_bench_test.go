package ctx

import "testing"

type benchStrArrDTO struct {
	Bool1   bool    `bind:"bool_1"`
	String1 string  `bind:"string_1"`
	Int1    int     `bind:"int_1"`
	Int2    int8    `bind:"ints_1.1"`
	Uint1   uint    `bind:"uint_1"`
	Float1  float64 `bind:"float_1"`
}

func BenchmarkBindStrArr(b *testing.B) {
	d := map[string][]string{
		"bool_1":   {"true"},
		"string_1": {"hello"},
		"int_1":    {"42"},
		"ints_1":   {"0", "7"},
		"uint_1":   {"99"},
		"float_1":  {"3.14"},
	}
	var fls []FieldLevel
	s := benchStrArrDTO{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fls = fls[:0]
		BindStrArr(d, &fls, s)
	}
}
