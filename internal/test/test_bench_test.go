package test

import "testing"

func BenchmarkDiffMessage_WithDesc(b *testing.B) {
	for range b.N {
		DiffMessage("actual-value", "expected-value", "some field description")
	}
}

func BenchmarkDiffMessage_NoDesc(b *testing.B) {
	for range b.N {
		DiffMessage("actual-value", "expected-value", "")
	}
}
