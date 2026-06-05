package runtime

import "testing"

func BenchmarkNodeID(b *testing.B) {
	for range b.N {
		NodeID()
	}
}
