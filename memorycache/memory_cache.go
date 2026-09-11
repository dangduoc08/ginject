package memorycache

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	numShards         = 256
	shardMask         = numShards - 1
	cleanupEvery      = 128
	cleanupBatch      = 64
	sweepEvery        = 5 * time.Second
	DefaultMaxEntries = 100_000
	evictSampleSize   = 8
)

var ErrEmptyKey = errors.New("memorycache: key must not be empty")

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

func (s *shard) admitLocked(now int64, key string, limit int) {
	if limit <= 0 || len(s.entriesByKey) < limit {
		return
	}
	if _, exists := s.entriesByKey[key]; exists {
		return
	}

	victim, found := "", false
	var victimDeadline int64
	n := 0
	for k, e := range s.entriesByKey {
		if e.expired(now) {
			delete(s.entriesByKey, k)
			if len(s.entriesByKey) < limit {
				return
			}
			continue
		}
		switch {
		case !found:
			victim, victimDeadline, found = k, e.expiresAt, true
		case e.expiresAt != 0 && (victimDeadline == 0 || e.expiresAt < victimDeadline):
			victim, victimDeadline = k, e.expiresAt
		}
		if n++; n >= evictSampleSize {
			break
		}
	}
	if found {
		delete(s.entriesByKey, victim)
	}
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

type Option func(*MemoryCache)

func WithMaxEntries(n int) Option {
	return func(mc *MemoryCache) {
		mc.maxEntries = n
	}
}

type MemoryCache struct {
	shards     [numShards]*shard
	done       chan struct{}
	maxEntries int
	shardLimit int
	stopOnce   sync.Once
	wg         sync.WaitGroup
}

func NewMemoryCache(opts ...Option) *MemoryCache {
	mc := &MemoryCache{done: make(chan struct{}), maxEntries: DefaultMaxEntries}
	for i := range mc.shards {
		mc.shards[i] = &shard{entriesByKey: make(map[string]entry)}
	}
	for _, opt := range opts {
		if opt != nil {
			opt(mc)
		}
	}
	mc.shardLimit = shardLimitFor(mc.maxEntries)
	mc.wg.Add(1)
	go mc.sweep()
	return mc
}

func shardLimitFor(maxEntries int) int {
	if maxEntries <= 0 {
		return 0
	}
	limit := (maxEntries + numShards - 1) / numShards
	if limit < 1 {
		limit = 1
	}
	return limit
}

func (mc *MemoryCache) Stop() {
	mc.stopOnce.Do(func() {
		close(mc.done)
	})
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
	s.admitLocked(now, key, mc.shardLimit)
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
	s.admitLocked(now, key, mc.shardLimit)
	s.entriesByKey[key] = entry{val: stored, expiresAt: deadline(now, ttl)}
	s.writes++
	if s.writes >= cleanupEvery {
		s.writes = 0
		s.evictLocked(now, cleanupBatch)
	}
	s.mu.Unlock()
	return true, nil
}

func (mc *MemoryCache) Mutate(_ context.Context, key string, fn func(old []byte, exists bool) (newVal []byte, ttl time.Duration)) ([]byte, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}
	now := time.Now().UnixNano()
	s := mc.shardOf(key)

	s.mu.Lock()
	defer s.mu.Unlock()

	e, exists := s.entriesByKey[key]
	if exists && e.expired(now) {
		exists = false
	}

	var old []byte
	if exists {
		old = make([]byte, len(e.val))
		copy(old, e.val)
	}

	newVal, ttl := fn(old, exists)

	stored := make([]byte, len(newVal))
	copy(stored, newVal)
	s.admitLocked(now, key, mc.shardLimit)
	s.entriesByKey[key] = entry{val: stored, expiresAt: deadline(now, ttl)}
	s.writes++
	if s.writes >= cleanupEvery {
		s.writes = 0
		s.evictLocked(now, cleanupBatch)
	}

	out := make([]byte, len(stored))
	copy(out, stored)
	return out, nil
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

	total := 0
	for _, s := range mc.shards {
		s.mu.RLock()
		total += len(s.entriesByKey)
		s.mu.RUnlock()
	}
	keys := make([]string, 0, total)
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
