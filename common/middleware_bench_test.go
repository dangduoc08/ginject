package common

import (
	"testing"
)

func BenchmarkInjectProvidersIntoRESTMiddlewares_ApplyAll(b *testing.B) {
	r := buildBenchREST(20)
	b.ResetTimer()
	for range b.N {
		m := &Middleware{}
		m.BindMiddleware(mockMiddlewareFn{})
		m.BindMiddleware(mockMiddlewareFn{})
		m.BindMiddleware(mockMiddlewareFn{})
		m.InjectProvidersIntoRESTMiddlewares(r, benchCB)
	}
}

func BenchmarkInjectProvidersIntoWSMiddlewares_ApplyAll(b *testing.B) {
	ws := buildBenchWS(20)
	b.ResetTimer()
	for range b.N {
		m := &Middleware{}
		m.BindMiddleware(mockMiddlewareFn{})
		m.BindMiddleware(mockMiddlewareFn{})
		m.BindMiddleware(mockMiddlewareFn{})
		m.InjectProvidersIntoWSMiddlewares(ws, benchCB)
	}
}
