package cache

import (
	"context"
	"fmt"
	"testing"
	"time"
)

var bctx = context.Background()

func BenchmarkGet_Hit(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	_ = svc.Set(bctx, "key", []byte("value"), 0)
	b.ResetTimer()
	for range b.N {
		svc.Get(bctx, "key")
	}
}

func BenchmarkGet_Miss(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	b.ResetTimer()
	for range b.N {
		svc.Get(bctx, "nonexistent")
	}
}

func BenchmarkSet(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	val := []byte("benchmark-value")
	b.ResetTimer()
	for i := range b.N {
		_ = svc.Set(bctx, fmt.Sprintf("k%d", i), val, 0)
	}
}

func BenchmarkSet_FixedKey(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	val := []byte("benchmark-value")
	b.ResetTimer()
	for range b.N {
		_ = svc.Set(bctx, "fixed", val, 0)
	}
}

func BenchmarkGet_Parallel(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	_ = svc.Set(bctx, "key", []byte("value"), 0)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			svc.Get(bctx, "key")
		}
	})
}

func BenchmarkSet_Parallel(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	val := []byte("benchmark-value")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = svc.Set(bctx, fmt.Sprintf("k%d", i), val, 0)
			i++
		}
	})
}

func BenchmarkMixed_Parallel(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	for i := 0; i < 1000; i++ {
		_ = svc.Set(bctx, fmt.Sprintf("k%d", i), []byte("v"), 0)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%4 == 0 {
				_ = svc.Set(bctx, fmt.Sprintf("k%d", i%1000), []byte("v"), 0)
			} else {
				svc.Get(bctx, fmt.Sprintf("k%d", i%1000))
			}
			i++
		}
	})
}

func BenchmarkGet_TTL_Expired(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	_ = svc.Set(bctx, "k", []byte("v"), time.Nanosecond)
	time.Sleep(time.Millisecond)
	b.ResetTimer()
	for range b.N {
		svc.Get(bctx, "k")
	}
}

func BenchmarkSetNX_Miss(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	b.ResetTimer()
	for i := range b.N {
		_, _ = svc.SetNX(bctx, fmt.Sprintf("k%d", i), []byte("v"), 0)
	}
}

func BenchmarkSetNX_Hit(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	_ = svc.Set(bctx, "key", []byte("v"), 0)
	b.ResetTimer()
	for range b.N {
		_, _ = svc.SetNX(bctx, "key", []byte("v"), 0)
	}
}

func BenchmarkSetNX_Parallel(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = svc.SetNX(bctx, fmt.Sprintf("k%d", i), []byte("v"), 0)
			i++
		}
	})
}

func BenchmarkKeys(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	for i := 0; i < 1000; i++ {
		_ = svc.Set(bctx, fmt.Sprintf("k%d", i), []byte("v"), 0)
	}
	b.ResetTimer()
	for range b.N {
		svc.Keys(bctx)
	}
}

func BenchmarkTTL_Hit(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	_ = svc.Set(bctx, "key", []byte("v"), time.Hour)
	b.ResetTimer()
	for range b.N {
		svc.TTL(bctx, "key")
	}
}

func BenchmarkTTL_Miss(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	b.ResetTimer()
	for range b.N {
		svc.TTL(bctx, "nonexistent")
	}
}

func BenchmarkDelete(b *testing.B) {
	svc := &CacheService{Engine: newMemoryCache()}
	b.ResetTimer()
	for i := range b.N {
		key := fmt.Sprintf("k%d", i)
		_ = svc.Set(bctx, key, []byte("v"), 0)
		_ = svc.Delete(bctx, key)
	}
}
