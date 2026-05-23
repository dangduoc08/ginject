package cache

import (
	"context"
	"time"

	"github.com/dangduoc08/ginject/core"
)

type CacheService struct {
	Backend Cache
}

func (cs CacheService) NewProvider() core.Provider {
	return cs
}

func (cs *CacheService) Get(ctx context.Context, key string) ([]byte, bool) {
	return cs.Backend.Get(ctx, key)
}

func (cs *CacheService) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return cs.Backend.Set(ctx, key, val, ttl)
}

func (cs *CacheService) Delete(ctx context.Context, key string) error {
	return cs.Backend.Delete(ctx, key)
}
