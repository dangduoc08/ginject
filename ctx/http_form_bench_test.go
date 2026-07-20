package ctx

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkHTTPContext_FormURLEncoded(b *testing.B) {
	c := newTestHTTPContext()
	var n int
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.form = nil
		c.header = nil
		c.Request = httptest.NewRequest("POST", "/", strings.NewReader("foo=bar&baz=qux"))
		c.Request.Header.Set("Content-Type", applicationXWWWFormUrlencoded)
		n += len(c.Form())
	}
	b.StopTimer()
	if n == 0 {
		b.Fatal("Form should not be empty")
	}
}

func BenchmarkForm_Bind(b *testing.B) {
	f := Form{"name": {"joe"}}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.Bind(formBindDTO{})
	}
}
