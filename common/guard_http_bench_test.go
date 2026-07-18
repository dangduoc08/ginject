package common

import (
	"testing"
)

func BenchmarkInjectProvidersIntoRESTGuards_ApplyAll(b *testing.B) {
	r := buildBenchREST(20)
	b.ResetTimer()
	for range b.N {
		g := &Guard{}
		g.BindGuard(mockGuarder{})
		g.BindGuard(mockGuarder{})
		g.BindGuard(mockGuarder{})
		g.InjectProvidersIntoRESTGuards(r, benchCB)
	}
}

func BenchmarkAsRESTGuard(b *testing.B) {
	guarder := mockGuarder{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsRESTGuard(guarder)
	}
}
