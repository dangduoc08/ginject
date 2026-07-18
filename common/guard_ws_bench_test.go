package common

import (
	"testing"
)

func BenchmarkInjectProvidersIntoWSGuards_ApplyAll(b *testing.B) {
	ws := buildBenchWS(20)
	b.ResetTimer()
	for range b.N {
		g := &Guard{}
		g.BindGuard(mockGuarder{})
		g.BindGuard(mockGuarder{})
		g.BindGuard(mockGuarder{})
		g.InjectProvidersIntoWSGuards(ws, benchCB)
	}
}

func BenchmarkAsWSGuard(b *testing.B) {
	guarder := mockGuarder{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsWSGuard(guarder)
	}
}
