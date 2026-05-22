package common

import (
	"testing"
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

func BenchmarkInjectProvidersIntoWSInterceptors_ApplyAll(b *testing.B) {
	ws := buildBenchWS(20)
	b.ResetTimer()
	for range b.N {
		ic := &Interceptor{}
		ic.BindInterceptor(mockInterceptable{})
		ic.BindInterceptor(mockInterceptable{})
		ic.BindInterceptor(mockInterceptable{})
		ic.InjectProvidersIntoWSInterceptors(ws, benchCB)
	}
}
