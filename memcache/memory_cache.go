package memcache

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	numShards    = 256
	shardMask    = numShards - 1
	cleanupEvery = 128
	cleanupBatch = 64
	sweepEvery   = 5 * time.Second
)

var ErrEmptyKey = errors.New("cache: key must not be empty")

type entry struct {
	val       []byte
	expiresAt int64
}

func (e entry) expired(now int64) bool {
	return e.expiresAt != 0 && now >= e.expiresAt
}

func deadline(now int64, ttl time.Duration) int64 {
	if ttl <= 0 {
		return 0
	}
	return now + int64(ttl)
}

type shard struct {
	mu           sync.RWMutex
	entriesByKey map[string]entry
	writes       int
}

func (s *shard) evictLocked(now int64, limit int) {
	n := 0
	for k, e := range s.entriesByKey {
		if e.expired(now) {
			delete(s.entriesByKey, k)
		}
		if n++; n >= limit {
			return
		}
	}
}

type MemoryCache struct {
	shards [numShards]*shard
	done   chan struct{}
	wg     sync.WaitGroup
}

func NewMemoryCache() *MemoryCache {
	mc := &MemoryCache{done: make(chan struct{})}
	for i := range mc.shards {
		mc.shards[i] = &shard{entriesByKey: make(map[string]entry)}
	}
	mc.wg.Add(1)
	go mc.sweep()
	return mc
}

func (mc *MemoryCache) Stop() {
	close(mc.done)
	mc.wg.Wait()
}

func (mc *MemoryCache) sweep() {
	defer mc.wg.Done()
	interval := sweepEvery / numShards
	if interval < time.Millisecond {
		interval = time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	idx := 0
	for {
		select {
		case <-ticker.C:
			mc.sweepShard(idx)
			idx = (idx + 1) & shardMask
		case <-mc.done:
			return
		}
	}
}

func (mc *MemoryCache) sweepShard(idx int) {
	s := mc.shards[idx]
	now := time.Now().UnixNano()
	s.mu.Lock()
	for k, e := range s.entriesByKey {
		if e.expired(now) {
			delete(s.entriesByKey, k)
		}
	}
	s.mu.Unlock()
}

func (mc *MemoryCache) shardOf(key string) *shard {
	return mc.shards[hashKey(key)&shardMask]
}

func (mc *MemoryCache) Get(_ context.Context, key string) ([]byte, bool) {
	if key == "" {
		return nil, false
	}
	now := time.Now().UnixNano()
	s := mc.shardOf(key)
	s.mu.RLock()
	e, ok := s.entriesByKey[key]
	s.mu.RUnlock()
	if !ok || e.expired(now) {
		return nil, false
	}
	out := make([]byte, len(e.val))
	copy(out, e.val)
	return out, true
}

func (mc *MemoryCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	if key == "" {
		return ErrEmptyKey
	}
	now := time.Now().UnixNano()
	stored := make([]byte, len(val))
	copy(stored, val)

	s := mc.shardOf(key)
	s.mu.Lock()
	s.entriesByKey[key] = entry{val: stored, expiresAt: deadline(now, ttl)}
	s.writes++
	if s.writes >= cleanupEvery {
		s.writes = 0
		s.evictLocked(now, cleanupBatch)
	}
	s.mu.Unlock()
	return nil
}

func (mc *MemoryCache) SetNX(_ context.Context, key string, val []byte, ttl time.Duration) (bool, error) {
	if key == "" {
		return false, ErrEmptyKey
	}
	now := time.Now().UnixNano()

	s := mc.shardOf(key)
	s.mu.Lock()
	if existing, exists := s.entriesByKey[key]; exists && !existing.expired(now) {
		s.mu.Unlock()
		return false, nil
	}
	stored := make([]byte, len(val))
	copy(stored, val)
	s.entriesByKey[key] = entry{val: stored, expiresAt: deadline(now, ttl)}
	s.mu.Unlock()
	return true, nil
}

func (mc *MemoryCache) Delete(_ context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	s := mc.shardOf(key)
	s.mu.Lock()
	delete(s.entriesByKey, key)
	s.mu.Unlock()
	return nil
}

func (mc *MemoryCache) Keys(_ context.Context) []string {
	now := time.Now().UnixNano()
	keys := make([]string, 0, 64)
	for _, s := range mc.shards {
		s.mu.RLock()
		for k, e := range s.entriesByKey {
			if !e.expired(now) {
				keys = append(keys, k)
			}
		}
		s.mu.RUnlock()
	}
	return keys
}

func (mc *MemoryCache) TTL(_ context.Context, key string) (time.Duration, bool) {
	if key == "" {
		return 0, false
	}
	now := time.Now().UnixNano()
	s := mc.shardOf(key)
	s.mu.RLock()
	e, ok := s.entriesByKey[key]
	s.mu.RUnlock()
	switch {
	case !ok || e.expired(now):
		return 0, false
	case e.expiresAt == 0:
		return 0, true
	default:
		return time.Duration(e.expiresAt - now), true
	}
}
