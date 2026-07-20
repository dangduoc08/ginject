package ctx

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkHTTPContext_Text(b *testing.B) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.ResponseWriter = httptest.NewRecorder()
		c.Text("hello world")
	}
}

func BenchmarkHTTPContext_JSON(b *testing.B) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	data := map[string]any{"foo": "bar", "n": 1}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.ResponseWriter = httptest.NewRecorder()
		c.JSON(data)
	}
}

func BenchmarkHTTPContext_SetID(b *testing.B) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.id = ""
		c.header = nil
		c.SetID()
	}
}
