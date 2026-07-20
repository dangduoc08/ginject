package common

import (
	"testing"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
)

func BenchmarkInjectProvidersIntoWSInterceptors_ApplyAll(b *testing.B) {
	ws := buildBenchWS(20)
	b.ResetTimer()
	for range b.N {
		ic := &Interceptor{}
		ic.BindInterceptor(mockWSInterceptable{})
		ic.BindInterceptor(mockWSInterceptable{})
		ic.BindInterceptor(mockWSInterceptable{})
		ic.InjectProvidersIntoWSInterceptors(ws, benchCB)
	}
}

func BenchmarkAsWSInterceptor(b *testing.B) {
	interceptable := mockWSInterceptable{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsWSInterceptor(interceptable)
	}
}

func BenchmarkBuildWSInterceptMiddleware(b *testing.B) {
	c := ctx.NewWSContext()
	c.Next = func() {}

	mw := BuildWSInterceptMiddleware("bench.event", func(*ctx.WSContext, *aggregation.Aggregation) any {
		return nil
	})

	b.ResetTimer()
	for range b.N {
		mw(c)
	}
}
