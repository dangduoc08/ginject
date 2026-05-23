package cache

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/testutils"
)

var ctx = context.Background()

func newSvc() *CacheService {
	return &CacheService{Backend: newMemoryCache()}
}

// --- Get ---

func TestGet_EmptyKey(t *testing.T) {
	svc := newSvc()
	val, ok := svc.Get(ctx, "")
	if ok || val != nil {
		t.Error(testutils.DiffMessage(ok, false, "empty key must return miss"))
	}
}

func TestGet_Missing(t *testing.T) {
	svc := newSvc()
	val, ok := svc.Get(ctx, "nonexistent")
	if ok || val != nil {
		t.Error(testutils.DiffMessage(ok, false, "missing key must return miss"))
	}
}

func TestGet_AfterSet(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("hello"), 0)
	val, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "key should exist after Set"))
	}
	if string(val) != "hello" {
		t.Error(testutils.DiffMessage(string(val), "hello", "value mismatch"))
	}
}

// --- Set ---

func TestSet_EmptyKey(t *testing.T) {
	svc := newSvc()
	err := svc.Set(ctx, "", []byte("v"), 0)
	if err == nil {
		t.Error(testutils.DiffMessage(err, ErrEmptyKey, "empty key must return error"))
	}
}

func TestSet_NilVal(t *testing.T) {
	svc := newSvc()
	err := svc.Set(ctx, "k", nil, 0)
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "nil val must not error"))
	}
	val, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "nil val key should exist"))
	}
	if len(val) != 0 {
		t.Error(testutils.DiffMessage(len(val), 0, "nil val should retrieve as empty"))
	}
}

func TestSet_EmptyVal(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte{}, 0)
	val, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "empty val key should exist"))
	}
	if len(val) != 0 {
		t.Error(testutils.DiffMessage(len(val), 0, "empty val should retrieve as empty"))
	}
}

func TestSet_Overwrite(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("first"), 0)
	_ = svc.Set(ctx, "k", []byte("second"), 0)
	val, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "key should exist after overwrite"))
	}
	if string(val) != "second" {
		t.Error(testutils.DiffMessage(string(val), "second", "overwritten value mismatch"))
	}
}

func TestSet_ValCopy_Mutation_After_Set(t *testing.T) {
	svc := newSvc()
	orig := []byte("mutable")
	_ = svc.Set(ctx, "k", orig, 0)
	orig[0] = 'X'
	val, _ := svc.Get(ctx, "k")
	if val[0] == 'X' {
		t.Error(testutils.DiffMessage(string(val), "mutable", "Set must copy val; caller mutation must not affect stored"))
	}
}

func TestGet_ValCopy_Mutation_After_Get(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("immutable"), 0)
	val, _ := svc.Get(ctx, "k")
	val[0] = 'X'
	val2, _ := svc.Get(ctx, "k")
	if val2[0] == 'X' {
		t.Error(testutils.DiffMessage(string(val2), "immutable", "Get must return a copy; mutation must not affect stored"))
	}
}

// --- Delete ---

func TestDelete_EmptyKey(t *testing.T) {
	svc := newSvc()
	err := svc.Delete(ctx, "")
	if err == nil {
		t.Error(testutils.DiffMessage(err, ErrEmptyKey, "empty key must return error"))
	}
}

func TestDelete_Removes(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("v"), 0)
	_ = svc.Delete(ctx, "k")
	_, ok := svc.Get(ctx, "k")
	if ok {
		t.Error(testutils.DiffMessage(ok, false, "key should be gone after Delete"))
	}
}

func TestDelete_Missing_NoError(t *testing.T) {
	svc := newSvc()
	err := svc.Delete(ctx, "nonexistent")
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "deleting missing key must not error"))
	}
}

// --- TTL ---

func TestTTL_ExpiredEntry_ReturnsMiss(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("v"), 10*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	_, ok := svc.Get(ctx, "k")
	if ok {
		t.Error(testutils.DiffMessage(ok, false, "expired entry must return miss"))
	}
}

func TestTTL_NotYetExpired_ReturnsHit(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("v"), 1*time.Hour)
	_, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "not-yet-expired entry must return hit"))
	}
}

func TestTTL_Zero_NeverExpires(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("v"), 0)
	time.Sleep(5 * time.Millisecond)
	_, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "ttl=0 must never expire"))
	}
}

func TestTTL_Negative_NeverExpires(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("v"), -1*time.Second)
	time.Sleep(5 * time.Millisecond)
	_, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "ttl<0 must never expire"))
	}
}

func TestTTL_OverwriteResetsExpiry(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "k", []byte("v1"), 10*time.Millisecond)
	_ = svc.Set(ctx, "k", []byte("v2"), 1*time.Hour)
	time.Sleep(20 * time.Millisecond)
	val, ok := svc.Get(ctx, "k")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "overwrite with longer TTL must not expire"))
	}
	if string(val) != "v2" {
		t.Error(testutils.DiffMessage(string(val), "v2", "overwritten value mismatch"))
	}
}

// --- Amortized cleanup ---

func TestCleanup_AmortizedRemovesExpired(t *testing.T) {
	svc := newSvc()
	shortTTL := 10 * time.Millisecond
	for i := 0; i < cleanupEvery*2; i++ {
		key := "expire-" + string(rune('a'+i%26))
		_ = svc.Set(ctx, key, []byte("v"), shortTTL)
	}
	time.Sleep(20 * time.Millisecond)
	_ = svc.Set(ctx, "trigger", []byte("cleanup"), 0)
	_, _ = svc.Get(ctx, "trigger")
}

// --- Large key / many keys ---

func TestLargeKey(t *testing.T) {
	svc := newSvc()
	key := string(bytes.Repeat([]byte("k"), 4096))
	err := svc.Set(ctx, key, []byte("v"), 0)
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "large key must not error"))
	}
	val, ok := svc.Get(ctx, key)
	if !ok || string(val) != "v" {
		t.Error(testutils.DiffMessage(ok, true, "large key must retrieve correctly"))
	}
}

func TestManyKeys(t *testing.T) {
	svc := newSvc()
	const n = 10000
	for i := 0; i < n; i++ {
		key := "key-" + string(rune(i))
		_ = svc.Set(ctx, key, []byte{byte(i % 256)}, 0)
	}
	for i := 0; i < n; i++ {
		key := "key-" + string(rune(i))
		_, ok := svc.Get(ctx, key)
		if !ok {
			t.Errorf("key %q should exist", key)
			return
		}
	}
}

// --- Concurrency ---

func TestConcurrent_ReadWrite(t *testing.T) {
	svc := newSvc()
	const goroutines = 100
	const ops = 500
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := "k" + string(rune(id%10))
			for j := 0; j < ops; j++ {
				if j%3 == 0 {
					_ = svc.Set(ctx, key, []byte{byte(j)}, 0)
				} else {
					svc.Get(ctx, key)
				}
			}
		}(i)
	}
	wg.Wait()
}

func TestConcurrent_TTLExpiry(t *testing.T) {
	svc := newSvc()
	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			key := "k" + string(rune(id))
			_ = svc.Set(ctx, key, []byte("v"), 5*time.Millisecond)
			time.Sleep(10 * time.Millisecond)
			svc.Get(ctx, key)
			_ = svc.Delete(ctx, key)
		}(i)
	}
	wg.Wait()
}

func TestConcurrent_DeleteRace(t *testing.T) {
	svc := newSvc()
	_ = svc.Set(ctx, "shared", []byte("v"), 0)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			svc.Get(ctx, "shared")
		}()
		go func() {
			defer wg.Done()
			_ = svc.Delete(ctx, "shared")
		}()
	}
	wg.Wait()
}
