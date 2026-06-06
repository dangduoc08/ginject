package memcache

import (
	"context"
	"errors"
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
	wg     sync.WaitGroup
}

var ErrEmptyKey = errors.New("cache: key must not be empty")

func NewMemoryCache() *memoryCache {
	mc := &memoryCache{done: make(chan struct{})}
	for i := range mc.shards {
		mc.shards[i] = &shard{items: make(map[string]item)}
	}
	mc.wg.Add(1)
	go mc.sweep()
	return mc
}

func (mc *memoryCache) Stop() {
	close(mc.done)
	mc.wg.Wait()
}

func (mc *memoryCache) sweep() {
	defer mc.wg.Done()
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
	now := time.Now().UnixNano()
	if it.expiresAt != 0 && now > it.expiresAt {
		s.mu.Lock()
		if cur, still := s.items[key]; still && cur.expiresAt != 0 && now > cur.expiresAt {
			delete(s.items, key)
		}
		s.mu.Unlock()
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
	now := time.Now()
	var expiresAt int64
	if ttl > 0 {
		expiresAt = now.Add(ttl).UnixNano()
	}
	stored := make([]byte, len(val))
	copy(stored, val)

	s := mc.shards[hashKey(key)&shardMask]
	s.mu.Lock()
	s.items[key] = item{val: stored, expiresAt: expiresAt}
	s.writes++
	if s.writes >= cleanupEvery {
		s.writes = 0
		nowNano := now.UnixNano()
		for k, it := range s.items {
			if it.expiresAt != 0 && nowNano > it.expiresAt {
				delete(s.items, k)
			}
		}
	}
	s.mu.Unlock()
	return nil
}

func (mc *memoryCache) SetNX(_ context.Context, key string, val []byte, ttl time.Duration) (bool, error) {
	if key == "" {
		return false, ErrEmptyKey
	}
	var expiresAt int64
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl).UnixNano()
	}

	s := mc.shards[hashKey(key)&shardMask]
	s.mu.Lock()
	it, exists := s.items[key]
	if exists && (it.expiresAt == 0 || time.Now().UnixNano() <= it.expiresAt) {
		s.mu.Unlock()
		return false, nil
	}
	stored := make([]byte, len(val))
	copy(stored, val)
	s.items[key] = item{val: stored, expiresAt: expiresAt}
	s.mu.Unlock()
	return true, nil
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

func (mc *memoryCache) Keys(_ context.Context) []string {
	now := time.Now().UnixNano()
	var total int
	for _, s := range mc.shards {
		s.mu.RLock()
		total += len(s.items)
		s.mu.RUnlock()
	}
	keys := make([]string, 0, total)
	for _, s := range mc.shards {
		s.mu.RLock()
		for k, it := range s.items {
			if it.expiresAt == 0 || now <= it.expiresAt {
				keys = append(keys, k)
			}
		}
		s.mu.RUnlock()
	}
	return keys
}

func (mc *memoryCache) TTL(_ context.Context, key string) (time.Duration, bool) {
	if key == "" {
		return 0, false
	}
	s := mc.shards[hashKey(key)&shardMask]
	s.mu.RLock()
	it, ok := s.items[key]
	s.mu.RUnlock()
	if !ok {
		return 0, false
	}
	if it.expiresAt == 0 {
		return 0, true
	}
	now := time.Now().UnixNano()
	remaining := it.expiresAt - now
	if remaining <= 0 {
		s.mu.Lock()
		if cur, still := s.items[key]; still && cur.expiresAt != 0 && now > cur.expiresAt {
			delete(s.items, key)
		}
		s.mu.Unlock()
		return 0, false
	}
	return time.Duration(remaining), true
}
