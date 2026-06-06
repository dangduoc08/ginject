package memcache

import (
	"context"
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
