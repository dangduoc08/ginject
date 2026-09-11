package memorycache

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestStop_IsIdempotentAndConcurrencySafe(t *testing.T) {
	mc := NewMemoryCache()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mc.Stop()
		}()
	}
	wg.Wait()

	mc.Stop()
}

func TestStop_TerminatesSweeperGoroutine(t *testing.T) {
	before := runtime.NumGoroutine()

	const instances = 20
	caches := make([]*MemoryCache, 0, instances)
	for i := 0; i < instances; i++ {
		caches = append(caches, NewMemoryCache())
	}

	for _, mc := range caches {
		if err := mc.Set(context.Background(), "k", []byte("v"), time.Minute); err != nil {
			t.Fatal(err)
		}
	}

	for _, mc := range caches {
		mc.Stop()
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+2 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Errorf("sweeper goroutines outlived Stop: %d before, %d after %d cache instances",
		before, runtime.NumGoroutine(), instances)
}

func TestWithMaxEntries_CapsShardGrowth(t *testing.T) {
	mc := NewMemoryCache(WithMaxEntries(numShards))
	defer mc.Stop()

	for i := 0; i < 20000; i++ {
		if err := mc.Set(context.Background(), fmt.Sprintf("k-%d", i), []byte("v"), 0); err != nil {
			t.Fatal(err)
		}
	}

	got := len(mc.Keys(context.Background()))
	if got > numShards {
		t.Errorf("a capped cache must not exceed its limit: %d entries for a cap of %d", got, numShards)
	}
	if got == 0 {
		t.Error("a capped cache must still hold entries")
	}
}

func TestWithMaxEntries_Zero_MeansUnlimited(t *testing.T) {
	mc := NewMemoryCache(WithMaxEntries(0))
	defer mc.Stop()

	const n = 5000
	for i := 0; i < n; i++ {
		if err := mc.Set(context.Background(), fmt.Sprintf("k-%d", i), []byte("v"), 0); err != nil {
			t.Fatal(err)
		}
	}

	if got := len(mc.Keys(context.Background())); got != n {
		t.Errorf("an explicitly unlimited cache must keep every entry, got %d want %d", got, n)
	}
}

func TestNewMemoryCache_DefaultsToBoundedCache(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()

	if mc.maxEntries != DefaultMaxEntries {
		t.Errorf("a new cache must be bounded by default, got maxEntries=%d", mc.maxEntries)
	}
	if mc.shardLimit <= 0 {
		t.Errorf("a bounded cache must have a positive per-shard limit, got %d", mc.shardLimit)
	}
}

func TestAdmitLocked_ReplacingExistingKey_DoesNotEvict(t *testing.T) {
	mc := NewMemoryCache(WithMaxEntries(numShards))
	defer mc.Stop()

	s := mc.shardOf("stable")
	s.mu.Lock()
	for i := 0; i < mc.shardLimit; i++ {
		s.entriesByKey[fmt.Sprintf("filler-%d", i)] = entry{val: []byte("v")}
	}
	s.entriesByKey["stable"] = entry{val: []byte("old")}
	before := len(s.entriesByKey)
	s.admitLocked(time.Now().UnixNano(), "stable", mc.shardLimit)
	after := len(s.entriesByKey)
	s.mu.Unlock()

	if after != before {
		t.Errorf("overwriting an existing key must not evict anything: %d -> %d", before, after)
	}
}
