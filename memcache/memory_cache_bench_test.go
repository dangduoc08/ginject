package memcache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkGet_Hit(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	if err := mc.Set(ctx, "key", []byte("benchmark-value-data-here"), 0); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.Get(ctx, "key")
		}
	})
}

func BenchmarkGet_Miss(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.Get(ctx, "nonexistent-key")
		}
	})
}

func BenchmarkSet_NoTTL(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	val := []byte("benchmark-value-data-here")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mc.Set(ctx, fmt.Sprintf("key-%d", i), val, 0)
	}
}

func BenchmarkSet_WithTTL(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	val := []byte("benchmark-value-data-here")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mc.Set(ctx, fmt.Sprintf("key-%d", i), val, time.Minute)
	}
}

func BenchmarkSetNX_New(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	val := []byte("benchmark-value-data-here")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mc.SetNX(ctx, fmt.Sprintf("key-%d", i), val, 0)
	}
}

func BenchmarkSetNX_Exists(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	val := []byte("benchmark-value-data-here")
	if err := mc.Set(ctx, "exists", val, 0); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = mc.SetNX(ctx, "exists", val, 0)
		}
	})
}

func BenchmarkDelete(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	val := []byte("v")
	for i := 0; i < b.N; i++ {
		_ = mc.Set(ctx, fmt.Sprintf("key-%d", i), val, 0)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mc.Delete(ctx, fmt.Sprintf("key-%d", i))
	}
}

func BenchmarkKeys_1000(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	for i := 0; i < 1000; i++ {
		if err := mc.Set(ctx, fmt.Sprintf("key-%d", i), []byte("v"), 0); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.Keys(ctx)
	}
}

func BenchmarkTTL_Hit(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	if err := mc.Set(ctx, "key", []byte("v"), time.Minute); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mc.TTL(ctx, "key")
		}
	})
}

func BenchmarkHashKey(b *testing.B) {
	key := "some-realistic-cache-key-value"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hashKey(key)
	}
}
