package cache

import (
	"testing"
)

func BenchmarkDLLUnshift(b *testing.B) {
	dll := NewDLL[string]()

	for i := 0; i < b.N; i++ {
		dll.unshift("boo")
	}
}

func BenchmarkDLLPush(b *testing.B) {
	dll := NewDLL[string]()

	for i := 0; i < b.N; i++ {
		dll.push("boo")
	}
}

func BenchmarkLFUCache2(b *testing.B) {
	cacheModule := New[string](CacheOpts{
		Strategy: LFU,
		Cap:      100,
	})

	cacheModule.Set("key_1", "value_1", -1)

	for i := 0; i < b.N; i++ {
		cacheModule.Get("key_1")
	}
}
