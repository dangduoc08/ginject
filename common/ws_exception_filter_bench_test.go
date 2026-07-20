package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

func BenchmarkInjectProvidersIntoWSExceptionFilters_ApplyAll(b *testing.B) {
	ws := buildBenchWS(20)
	b.ResetTimer()
	for range b.N {
		e := &ExceptionFilter{}
		e.BindExceptionFilter(mockWSExFilter{})
		e.BindExceptionFilter(mockWSExFilter{})
		e.BindExceptionFilter(mockWSExFilter{})
		e.InjectProvidersIntoWSExceptionFilters(ws, benchCB)
	}
}

func BenchmarkAsWSExceptionFilter(b *testing.B) {
	exceptionFilterable := mockWSExFilter{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsWSExceptionFilter(exceptionFilterable)
	}
}

func BenchmarkBuildWSCatchMiddleware(b *testing.B) {
	c := ctx.NewWSContext()
	c.Next = func() {}

	mw := BuildWSCatchMiddleware("bench.event", []WSCatch{
		func(*ctx.WSContext, *exception.Exception) {},
	})

	b.ResetTimer()
	for range b.N {
		mw(c)
	}
}
