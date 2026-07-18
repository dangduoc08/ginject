package ctx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/broker"
)

func BenchmarkSetID_FromHeader(b *testing.B) {
	c := &HTTPContext{Broker: broker.New()}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(RequestID, "bench-request-id")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.id = ""
		c.Request = r
		c.SetID()
	}
}

func BenchmarkSetID_Generated(b *testing.B) {
	c := &HTTPContext{Broker: broker.New()}
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.id = ""
		c.Request = r
		c.SetID()
	}
}
