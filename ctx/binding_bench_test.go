package ctx

import "testing"

type benchBindDTO struct {
	Name    string       `bind:"name"`
	Age     int          `bind:"age"`
	Active  bool         `bind:"active"`
	Tags    []string     `bind:"tags"`
	Address benchAddrDTO `bind:"address"`
}

type benchAddrDTO struct {
	Street string `bind:"street"`
	City   string `bind:"city"`
}

func benchBindData() map[string]any {
	return map[string]any{
		"name":   "John Doe",
		"age":    float64(30),
		"active": true,
		"tags":   []any{"a", "b", "c"},
		"address": map[string]any{
			"street": "Main St",
			"city":   "Metropolis",
		},
	}
}

func BenchmarkBindStruct(b *testing.B) {
	data := benchBindData()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BindStruct(data, &[]FieldLevel{}, benchBindDTO{}, "", "")
	}
}

type benchPtrDTO struct {
	Name *string `bind:"name"`
	Age  *int    `bind:"age"`
}

func BenchmarkBindStruct_PointerFields(b *testing.B) {
	data := map[string]any{"name": "John Doe", "age": float64(30)}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BindStruct(data, &[]FieldLevel{}, benchPtrDTO{}, "", "")
	}
}
