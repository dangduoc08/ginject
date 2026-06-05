package cache

import (
	"context"
	"time"

	"github.com/dangduoc08/ginject/core"
)

type CacheService struct {
	Engine Cache
}

func (cs CacheService) NewProvider() core.Provider {
	return cs
}

func (cs *CacheService) Get(ctx context.Context, key string) ([]byte, bool) {
	return cs.Engine.Get(ctx, key)
}

func (cs *CacheService) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return cs.Engine.Set(ctx, key, val, ttl)
}

func (cs *CacheService) SetNX(ctx context.Context, key string, val []byte, ttl time.Duration) (bool, error) {
	return cs.Engine.SetNX(ctx, key, val, ttl)
}

func (cs *CacheService) Delete(ctx context.Context, key string) error {
	return cs.Engine.Delete(ctx, key)
}

func (cs *CacheService) Keys(ctx context.Context) []string {
	return cs.Engine.Keys(ctx)
}

func (cs *CacheService) TTL(ctx context.Context, key string) (time.Duration, bool) {
	return cs.Engine.TTL(ctx, key)
}
