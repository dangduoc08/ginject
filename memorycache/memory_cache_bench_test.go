package memorycache

import (
	"context"
	"fmt"
	"strconv"
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
	keys := benchKeys(b.N)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mc.Set(ctx, keys[i], val, 0)
	}
}

func BenchmarkSet_WithTTL(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	val := []byte("benchmark-value-data-here")
	keys := benchKeys(b.N)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mc.Set(ctx, keys[i], val, time.Minute)
	}
}

func BenchmarkSetNX_New(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()
	val := []byte("benchmark-value-data-here")
	keys := benchKeys(b.N)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mc.SetNX(ctx, keys[i], val, 0)
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

// BenchmarkGetSetParallel is the benchmark sharding exists for: concurrent
// access spread across shards, where lock contention and cache-line sharing
// between adjacent shards dominate.
func BenchmarkGetSetParallel(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()

	ctx := context.Background()
	keys := make([]string, 4096)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		_ = mc.Set(ctx, keys[i], []byte("value-payload"), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			k := keys[i&4095]
			if i&7 == 0 {
				_ = mc.Set(ctx, k, []byte("value-payload"), time.Hour)
			} else {
				mc.Get(ctx, k)
			}
			i++
		}
	})
}

func BenchmarkGetParallel(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()

	ctx := context.Background()
	keys := make([]string, 4096)
	for i := range keys {
		keys[i] = fmt.Sprintf("key-%d", i)
		_ = mc.Set(ctx, keys[i], []byte("value-payload"), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			mc.Get(ctx, keys[i&4095])
			i++
		}
	})
}

func BenchmarkMutate(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()

	ctx := context.Background()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = mc.Mutate(ctx, "counter", func(old []byte, _ bool) ([]byte, time.Duration) {
			return []byte("payload"), time.Hour
		})
	}
}

func BenchmarkKeys_100000(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()

	ctx := context.Background()
	for i := 0; i < 100000; i++ {
		_ = mc.Set(ctx, fmt.Sprintf("key-%d", i), []byte("v"), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = mc.Keys(ctx)
	}
}

func BenchmarkNewMemoryCache(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		mc := NewMemoryCache()
		mc.Stop()
	}
}

func benchKeys(n int) []string {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "key-" + strconv.Itoa(i)
	}
	return keys
}

// BenchmarkGetParallel_LongKeys uses realistically long composite keys, where
// the shard hash is a meaningful share of the lookup cost.
func BenchmarkGetParallel_LongKeys(b *testing.B) {
	mc := NewMemoryCache()
	defer mc.Stop()

	ctx := context.Background()
	keys := make([]string, 4096)
	for i := range keys {
		keys[i] = "tenant:acme-corp:user:session:" + strconv.Itoa(i) + ":profile-cache-entry"
		_ = mc.Set(ctx, keys[i], []byte("value-payload"), time.Hour)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			mc.Get(ctx, keys[i&4095])
			i++
		}
	})
}
