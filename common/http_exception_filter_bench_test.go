package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

func BenchmarkInjectProvidersIntoHTTPExceptionFilters_ApplyAll(b *testing.B) {
	r := buildBenchHTTP(20)
	b.ResetTimer()
	for range b.N {
		e := &ExceptionFilter{}
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.InjectProvidersIntoHTTPExceptionFilters(r, benchCB)
	}
}

func BenchmarkAsHTTPExceptionFilter(b *testing.B) {
	exceptionFilterable := mockExFilter{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsHTTPExceptionFilter(exceptionFilterable)
	}
}

func BenchmarkBuildHTTPCatchMiddleware(b *testing.B) {
	c := ctx.NewHTTPContext()
	c.Next = func() {}

	mw := BuildHTTPCatchMiddleware("bench.event", []HTTPCatch{
		func(*ctx.HTTPContext, *exception.Exception) {},
	})

	b.ResetTimer()
	for range b.N {
		mw(c)
	}
}
