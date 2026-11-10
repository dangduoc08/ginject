package crypto

import (
	"sync/atomic"
	"testing"
)

var parallelSink atomic.Pointer[string]

func BenchmarkUUID(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		UUID()
	}
}

func BenchmarkUUID_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var local string
		for pb.Next() {
			local = UUID()
		}
		parallelSink.Store(&local)
	})
}
