package memorycache

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestMemoryCache_GetSet(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "key", []byte("val"), 0); err != nil {
		t.Fatal(err)
	}
	got, ok := mc.Get(ctx, "key")
	if !ok {
		t.Error(test.DiffMessage(ok, true, "Get after Set: ok"))
	}
	if string(got) != "val" {
		t.Error(test.DiffMessage(string(got), "val", "Get after Set: value"))
	}
}

func TestMemoryCache_GetMiss(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	got, ok := mc.Get(ctx, "missing")
	if ok || got != nil {
		t.Error(test.DiffMessage(ok, false, "Get miss"))
	}
}

func TestMemoryCache_EmptyKey(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if _, ok := mc.Get(ctx, ""); ok {
		t.Error(test.DiffMessage(true, false, "Get empty key"))
	}
	if err := mc.Set(ctx, "", nil, 0); err != ErrEmptyKey {
		t.Error(test.DiffMessage(err, ErrEmptyKey, "Set empty key"))
	}
	if _, err := mc.SetNX(ctx, "", nil, 0); err != ErrEmptyKey {
		t.Error(test.DiffMessage(err, ErrEmptyKey, "SetNX empty key"))
	}
	if err := mc.Delete(ctx, ""); err != ErrEmptyKey {
		t.Error(test.DiffMessage(err, ErrEmptyKey, "Delete empty key"))
	}
	if _, ok := mc.TTL(ctx, ""); ok {
		t.Error(test.DiffMessage(true, false, "TTL empty key"))
	}
}

func TestMemoryCache_TTLExpiry(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "exp", []byte("v"), 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if _, ok := mc.Get(ctx, "exp"); !ok {
		t.Error(test.DiffMessage(false, true, "Get before expiry"))
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := mc.Get(ctx, "exp"); ok {
		t.Error(test.DiffMessage(true, false, "Get after expiry"))
	}
}

func TestMemoryCache_NoExpiry(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "persistent", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	dur, ok := mc.TTL(ctx, "persistent")
	if !ok {
		t.Error(test.DiffMessage(false, true, "TTL ok for no-expiry key"))
	}
	if dur != 0 {
		t.Error(test.DiffMessage(dur, time.Duration(0), "TTL value for no-expiry key"))
	}
}

func TestMemoryCache_TTLWithExpiry(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	ttl := 500 * time.Millisecond
	if err := mc.Set(ctx, "k", []byte("v"), ttl); err != nil {
		t.Fatal(err)
	}
	remaining, ok := mc.TTL(ctx, "k")
	if !ok {
		t.Error(test.DiffMessage(false, true, "TTL ok"))
	}
	if remaining <= 0 || remaining > ttl {
		t.Error(test.DiffMessage(remaining, "0 < remaining <= 500ms", "TTL remaining in range"))
	}
}

func TestMemoryCache_TTLExpired(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "short", []byte("v"), 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	if _, ok := mc.TTL(ctx, "short"); ok {
		t.Error(test.DiffMessage(true, false, "TTL after expiry"))
	}
}

func TestMemoryCache_TTLMissing(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if _, ok := mc.TTL(ctx, "nope"); ok {
		t.Error(test.DiffMessage(true, false, "TTL missing key"))
	}
}

func TestMemoryCache_SetNX_New(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	set, err := mc.SetNX(ctx, "nx", []byte("first"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !set {
		t.Error(test.DiffMessage(false, true, "SetNX first write"))
	}
}

func TestMemoryCache_SetNX_Exists(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "nx", []byte("first"), 0); err != nil {
		t.Fatal(err)
	}
	set, err := mc.SetNX(ctx, "nx", []byte("second"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if set {
		t.Error(test.DiffMessage(true, false, "SetNX with existing key"))
	}
	got, _ := mc.Get(ctx, "nx")
	if string(got) != "first" {
		t.Error(test.DiffMessage(string(got), "first", "SetNX does not overwrite existing value"))
	}
}

func TestMemoryCache_SetNX_AfterExpiry(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "nx", []byte("old"), 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)
	set, err := mc.SetNX(ctx, "nx", []byte("new"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !set {
		t.Error(test.DiffMessage(false, true, "SetNX after expiry"))
	}
	got, _ := mc.Get(ctx, "nx")
	if string(got) != "new" {
		t.Error(test.DiffMessage(string(got), "new", "SetNX value after expiry"))
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "del", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	if err := mc.Delete(ctx, "del"); err != nil {
		t.Fatal(err)
	}
	if _, ok := mc.Get(ctx, "del"); ok {
		t.Error(test.DiffMessage(true, false, "Get after Delete"))
	}
}

func TestMemoryCache_DeleteMissing(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Delete(ctx, "ghost"); err != nil {
		t.Error(test.DiffMessage(err, nil, "Delete missing key should not error"))
	}
}

func TestMemoryCache_Keys(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	for _, kv := range []struct{ k, v string }{{"a", "1"}, {"b", "2"}} {
		if err := mc.Set(ctx, kv.k, []byte(kv.v), 0); err != nil {
			t.Fatal(err)
		}
	}
	if err := mc.Set(ctx, "expired", []byte("3"), 20*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(40 * time.Millisecond)

	keys := mc.Keys(ctx)
	got := make(map[string]bool, len(keys))
	for _, k := range keys {
		got[k] = true
	}
	if !got["a"] || !got["b"] {
		t.Error(test.DiffMessage(got, map[string]bool{"a": true, "b": true}, "Keys includes live keys"))
	}
	if got["expired"] {
		t.Error(test.DiffMessage(got["expired"], false, "Keys excludes expired key"))
	}
}

func TestMemoryCache_CopySemantics_Set(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	original := []byte("hello")
	if err := mc.Set(ctx, "copy", original, 0); err != nil {
		t.Fatal(err)
	}
	original[0] = 'X'

	got, _ := mc.Get(ctx, "copy")
	if string(got) != "hello" {
		t.Error(test.DiffMessage(string(got), "hello", "Set stores a copy, mutation does not affect stored value"))
	}
}

func TestMemoryCache_CopySemantics_Get(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "copy", []byte("hello"), 0); err != nil {
		t.Fatal(err)
	}
	got, _ := mc.Get(ctx, "copy")
	got[0] = 'X'

	got2, _ := mc.Get(ctx, "copy")
	if string(got2) != "hello" {
		t.Error(test.DiffMessage(string(got2), "hello", "Get returns a copy, mutation does not affect stored value"))
	}
}

func TestMemoryCache_NilValue(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()
	ctx := context.Background()

	if err := mc.Set(ctx, "nil", nil, 0); err != nil {
		t.Fatal(err)
	}
	got, ok := mc.Get(ctx, "nil")
	if !ok {
		t.Error(test.DiffMessage(false, true, "Get ok for nil-value key"))
	}
	if len(got) != 0 {
		t.Error(test.DiffMessage(got, []byte(nil), "Get nil value"))
	}
}

func TestMemoryCache_Mutate_NewKey(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()

	got, err := mc.Mutate(context.Background(), "counter", func(old []byte, exists bool) ([]byte, time.Duration) {
		if exists {
			t.Error(test.DiffMessage(exists, false, "a missing key must report exists=false"))
		}
		if old != nil {
			t.Error(test.DiffMessage(old, nil, "a missing key must pass nil old value"))
		}
		return []byte("1"), 0
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "1" {
		t.Error(test.DiffMessage(string(got), "1", "Mutate must return the stored value"))
	}

	stored, ok := mc.Get(context.Background(), "counter")
	if !ok || string(stored) != "1" {
		t.Error(test.DiffMessage(string(stored), "1", "Mutate must persist the new value"))
	}
}

func TestMemoryCache_Mutate_ExistingKey(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()

	if err := mc.Set(context.Background(), "k", []byte("old"), 0); err != nil {
		t.Fatal(err)
	}

	_, err := mc.Mutate(context.Background(), "k", func(old []byte, exists bool) ([]byte, time.Duration) {
		if !exists {
			t.Error(test.DiffMessage(exists, true, "an existing key must report exists=true"))
		}
		if string(old) != "old" {
			t.Error(test.DiffMessage(string(old), "old", "Mutate must pass the current value"))
		}
		return []byte("new"), 0
	})
	if err != nil {
		t.Fatal(err)
	}

	got, _ := mc.Get(context.Background(), "k")
	if string(got) != "new" {
		t.Error(test.DiffMessage(string(got), "new", "Mutate must overwrite the value"))
	}
}

func TestMemoryCache_Mutate_ExpiredTreatedAsMissing(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()

	if err := mc.Set(context.Background(), "k", []byte("stale"), time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)

	_, err := mc.Mutate(context.Background(), "k", func(old []byte, exists bool) ([]byte, time.Duration) {
		if exists {
			t.Error(test.DiffMessage(exists, false, "an expired entry must be reported as missing"))
		}
		return []byte("fresh"), 0
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMemoryCache_Mutate_ReturnedSliceIsACopy(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()

	got, err := mc.Mutate(context.Background(), "k", func([]byte, bool) ([]byte, time.Duration) {
		return []byte("value"), 0
	})
	if err != nil {
		t.Fatal(err)
	}

	got[0] = 'X'
	stored, _ := mc.Get(context.Background(), "k")
	if string(stored) != "value" {
		t.Error(test.DiffMessage(string(stored), "value", "mutating the returned slice must not corrupt the stored entry"))
	}
}

func TestMemoryCache_Mutate_EmptyKey(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()

	if _, err := mc.Mutate(context.Background(), "", func([]byte, bool) ([]byte, time.Duration) {
		t.Error("fn must not run for an empty key")
		return nil, 0
	}); err != ErrEmptyKey {
		t.Error(test.DiffMessage(err, ErrEmptyKey, "an empty key must be rejected"))
	}
}

func TestMemoryCache_ConcurrentMixedOperations(t *testing.T) {
	mc := NewMemoryCache()
	defer mc.Stop()

	const workers = 16
	const iterations = 300

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			ctx := context.Background()
			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("k-%d-%d", w, i%20)
				_ = mc.Set(ctx, key, []byte("v"), time.Minute)
				mc.Get(ctx, key)
				_, _ = mc.SetNX(ctx, key+"-nx", []byte("v"), time.Minute)
				_, _ = mc.Mutate(ctx, key+"-m", func(old []byte, _ bool) ([]byte, time.Duration) {
					return append(append([]byte{}, old...), 'x'), time.Minute
				})
				mc.TTL(ctx, key)
				_ = mc.Delete(ctx, key)
			}
		}(w)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = mc.Keys(context.Background())
		}
	}()

	wg.Wait()
}

func TestHashKey_DistributesAcrossShards(t *testing.T) {
	const keys = numShards * 40

	counts := map[uint64]int{}
	for i := 0; i < keys; i++ {
		counts[hashKey("tenant:acme:user:session:"+strconv.Itoa(i))&shardMask]++
	}

	maxCount := 0
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}
	empty := numShards - len(counts)

	if empty > numShards/10 {
		t.Error(test.DiffMessage(empty, 0, "too many shards received no key; the hash is not spreading keys"))
	}
	if maxCount > 40*6 {
		t.Error(test.DiffMessage(maxCount, 40, "one shard absorbed far more keys than its fair share"))
	}
}

func TestHashKey_Deterministic(t *testing.T) {
	const key = "tenant:acme:user:session:42"

	first := hashKey(key)
	for i := 0; i < 100; i++ {
		if got := hashKey(key); got != first {
			t.Fatal(test.DiffMessage(got, first, "hashKey must be stable for the same key within a process"))
		}
	}
}

func TestHashKey_EmptyKeyDoesNotPanic(t *testing.T) {
	_ = hashKey("")
}

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
