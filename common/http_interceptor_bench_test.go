package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
)

func BenchmarkInjectProvidersIntoRESTInterceptors_ApplyAll(b *testing.B) {
	r := buildBenchREST(20)
	b.ResetTimer()
	for range b.N {
		ic := &Interceptor{}
		ic.BindInterceptor(mockInterceptable{})
		ic.BindInterceptor(mockInterceptable{})
		ic.BindInterceptor(mockInterceptable{})
		ic.InjectProvidersIntoRESTInterceptors(r, benchCB)
	}
}

func BenchmarkAsRESTInterceptor(b *testing.B) {
	interceptable := mockInterceptable{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsRESTInterceptor(interceptable)
	}
}

func BenchmarkBuildHTTPInterceptMiddleware(b *testing.B) {
	c := ctx.NewHTTPContext()
	c.Init(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	c.Next = func() {}

	mw := BuildHTTPInterceptMiddleware("bench.key", func(*ctx.HTTPContext, *aggregation.Aggregation) any {
		return nil
	})

	b.ResetTimer()
	for range b.N {
		mw(c)
	}
}
