package crypto

import "testing"

var parallelSink string

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
		parallelSink = local
	})
}
