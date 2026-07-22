package common

import (
	"testing"
)

func BenchmarkInjectProvidersIntoHTTPMiddlewares_ApplyAll(b *testing.B) {
	r := buildBenchHTTP(20)
	b.ResetTimer()
	for range b.N {
		m := &Middleware{}
		m.BindMiddleware(mockMiddlewareFn{})
		m.BindMiddleware(mockMiddlewareFn{})
		m.BindMiddleware(mockMiddlewareFn{})
		m.InjectProvidersIntoHTTPMiddlewares(r, benchCB)
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
