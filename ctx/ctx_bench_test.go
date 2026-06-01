package ctx

import (
	"testing"
)

func BenchmarkGetTagParams_Single(b *testing.B) {
	v := "field_name"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParams(v)
	}
}

func BenchmarkGetTagParams_Multiple(b *testing.B) {
	v := "field_name, required, min=1, max=100"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParams(v)
	}
}

func BenchmarkGetTagParamIndex_WithDot(b *testing.B) {
	v := "integers_1.3"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParamIndex(v)
	}
}

func BenchmarkGetTagParamIndex_NoDot(b *testing.B) {
	v := "field_name"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetTagParamIndex(v)
	}
}

func BenchmarkResolveWSEventname(b *testing.B) {
	e := "CHAT_HELLO_WORLD"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ResolveWSEventname(e)
	}
}

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

func BenchmarkBodyGet_Shallow(b *testing.B) {
	body := Body{"key": "value"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body.Get("key")
	}
}

func BenchmarkBodyGet_Deep(b *testing.B) {
	body := Body{
		"a": map[string]any{
			"b": map[string]any{
				"c": map[string]any{
					"d": "value",
				},
			},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body.Get("a.b.c.d")
	}
}
