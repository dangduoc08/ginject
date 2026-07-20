package ctx

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkHTTPContext_Header(b *testing.B) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Foo", "bar")
	var n int
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.header = nil
		n += len(c.Header())
	}
	b.StopTimer()
	if n == 0 {
		b.Fatal("Header should not be empty")
	}
}

func BenchmarkHeader_Bind(b *testing.B) {
	h := Header{}
	h.Set("Authorization", "Bearer token")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Bind(headerBindDTO{})
	}
}
