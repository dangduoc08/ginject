package ctx

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkHTTPContext_Query(b *testing.B) {
	c := newTestHTTPContext()
	c.Request = httptest.NewRequest("GET", "/?foo=bar&baz=qux", nil)
	var n int
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.query = nil
		n += len(c.Query())
	}
	b.StopTimer()
	if n == 0 {
		b.Fatal("Query should not be empty")
	}
}

func BenchmarkQuery_Bind(b *testing.B) {
	q := Query{"name": {"joe"}, "age": {"30"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Bind(queryBindDTO{})
	}
}
