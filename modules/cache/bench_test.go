package cache

import (
	"testing"
)

func BenchmarkDLLUnshift(b *testing.B) {
	dll := newDLL[string]()

	for i := 0; i < b.N; i++ {
		dll.unshift("boo")
	}
}

func BenchmarkDLLPush(b *testing.B) {
	dll := newDLL[string]()

	for i := 0; i < b.N; i++ {
		dll.push("boo")
	}
}
