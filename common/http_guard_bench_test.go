package common

import (
	"testing"
)

func BenchmarkInjectProvidersIntoHTTPGuards_ApplyAll(b *testing.B) {
	r := buildBenchHTTP(20)
	b.ResetTimer()
	for range b.N {
		g := &Guard{}
		g.BindGuard(mockGuarder{})
		g.BindGuard(mockGuarder{})
		g.BindGuard(mockGuarder{})
		g.InjectProvidersIntoHTTPGuards(r, benchCB)
	}
}

func BenchmarkAsHTTPGuard(b *testing.B) {
	guarder := mockGuarder{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsHTTPGuard(guarder)
	}
}
