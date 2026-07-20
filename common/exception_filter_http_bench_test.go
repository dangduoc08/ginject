package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

func BenchmarkInjectProvidersIntoRESTExceptionFilters_ApplyAll(b *testing.B) {
	r := buildBenchREST(20)
	b.ResetTimer()
	for range b.N {
		e := &ExceptionFilter{}
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.InjectProvidersIntoRESTExceptionFilters(r, benchCB)
	}
}

func BenchmarkAsRESTExceptionFilter(b *testing.B) {
	exceptionFilterable := mockExFilter{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsRESTExceptionFilter(exceptionFilterable)
	}
}

func BenchmarkBuildHTTPCatchMiddleware(b *testing.B) {
	c := ctx.NewHTTPContext()
	c.Next = func() {}

	mw := BuildHTTPCatchMiddleware("bench.event", []RESTCatch{
		func(*ctx.HTTPContext, *exception.Exception) {},
	})

	b.ResetTimer()
	for range b.N {
		mw(c)
	}
}
