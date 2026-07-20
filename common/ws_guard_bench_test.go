package common

import (
	"testing"
)

func BenchmarkInjectProvidersIntoWSGuards_ApplyAll(b *testing.B) {
	ws := buildBenchWS(20)
	b.ResetTimer()
	for range b.N {
		g := &Guard{}
		g.BindGuard(mockWSGuarder{})
		g.BindGuard(mockWSGuarder{})
		g.BindGuard(mockWSGuarder{})
		g.InjectProvidersIntoWSGuards(ws, benchCB)
	}
}

func BenchmarkAsWSGuard(b *testing.B) {
	guarder := mockWSGuarder{}
	b.ResetTimer()
	for range b.N {
		_, _ = AsWSGuard(guarder)
	}
}
