package ctx

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkJSON_WriteData(b *testing.B) {
	data := []any{map[string]any{"foo": "bar", "n": 1}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := &JSON{responseWriter: httptest.NewRecorder(), data: data}
		j.WriteData(200)
	}
}

func BenchmarkText_WriteDataWithArgs(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx := &Text{responseWriter: httptest.NewRecorder(), data: "hello %s", args: []any{"joe"}}
		tx.WriteData(200)
	}
}
