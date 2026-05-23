package cache

import (
	"context"
	"sync"
	"time"
)

const (
	numShards    = 64
	shardMask    = numShards - 1
	cleanupEvery = 100
)

type item struct {
	val       []byte
	expiresAt int64 // unix nanoseconds; 0 = no expiry
}

type shard struct {
	mu     sync.RWMutex
	items  map[string]item
	writes int
}

type memoryCache struct {
	shards [numShards]*shard
	done   chan struct{}
}

func NewMemoryCache() *memoryCache { return newMemoryCache() }

func newMemoryCache() *memoryCache {
	mc := &memoryCache{done: make(chan struct{})}
	for i := range mc.shards {
		mc.shards[i] = &shard{items: make(map[string]item)}
	}
	go mc.sweep()
	return mc
}

func (mc *memoryCache) Stop() {
	close(mc.done)
}

func (mc *memoryCache) sweep() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			for _, s := range mc.shards {
				s.mu.Lock()
				for k, it := range s.items {
					if it.expiresAt != 0 && now > it.expiresAt {
						delete(s.items, k)
					}
				}
				s.mu.Unlock()
			}
		case <-mc.done:
			return
		}
	}
}

func hashKey(key string) uint32 {
	const (
		offset32 = uint32(2166136261)
		prime32  = uint32(16777619)
	)
	h := offset32
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= prime32
	}
	return h
}

func (mc *memoryCache) Get(_ context.Context, key string) ([]byte, bool) {
	if key == "" {
		return nil, false
	}
	s := mc.shards[hashKey(key)&shardMask]
	s.mu.RLock()
	it, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if it.expiresAt != 0 && time.Now().UnixNano() > it.expiresAt {
		return nil, false
	}
	out := make([]byte, len(it.val))
	copy(out, it.val)
	return out, true
}

func (mc *memoryCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) error {
	if key == "" {
		return ErrEmptyKey
	}
	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().UnixNano() + int64(ttl)
	}
	stored := make([]byte, len(val))
	copy(stored, val)

	s := mc.shards[hashKey(key)&shardMask]
	s.mu.Lock()
	s.items[key] = item{val: stored, expiresAt: expiresAt}
	s.writes++
	if s.writes >= cleanupEvery {
		s.writes = 0
		now := time.Now().UnixNano()
		for k, it := range s.items {
			if it.expiresAt != 0 && now > it.expiresAt {
				delete(s.items, k)
			}
		}
	}
	s.mu.Unlock()
	return nil
}

func (mc *memoryCache) Delete(_ context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}
	s := mc.shards[hashKey(key)&shardMask]
	s.mu.Lock()
	delete(s.items, key)
	s.mu.Unlock()
	return nil
}
