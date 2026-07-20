package ctx

import (
	"reflect"
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

func BenchmarkBindArray_Int(b *testing.B) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([]int{})
	arr := []any{float64(1), float64(2), float64(3), float64(4), float64(5)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindArray(arr, fls, typ, "", "")
	}
}

func BenchmarkBindArray_Struct(b *testing.B) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf([]fnArrStructDTO{})
	arr := []any{
		map[string]any{"name": "joe"},
		map[string]any{"name": "ana"},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindArray(arr, fls, typ, "", "")
	}
}

func BenchmarkBindMap_Int(b *testing.B) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]int{})
	obj := map[string]any{"a": float64(1), "b": float64(2), "c": float64(3)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindMap(obj, fls, typ, "", "")
	}
}

func BenchmarkBindMap_Struct(b *testing.B) {
	fls := &[]FieldLevel{}
	typ := reflect.TypeOf(map[string]fnArrStructDTO{})
	obj := map[string]any{"a": map[string]any{"name": "joe"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bindMap(obj, fls, typ, "", "")
	}
}

func BenchmarkToJSONBuffer(b *testing.B) {
	data := map[string]any{"foo": "bar", "n": 1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = toJSONBuffer(data)
	}
}
