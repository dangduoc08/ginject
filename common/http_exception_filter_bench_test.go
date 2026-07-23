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

func BenchmarkRunHTTPCatchChain(b *testing.B) {
	c := ctx.NewHTTPContext()
	catchFns := []HTTPCatch{
		func(*ctx.HTTPContext, *exception.Exception) {},
	}

	b.ResetTimer()
	for range b.N {
		RunHTTPCatchChain(c, catchFns, "boom")
	}
}
