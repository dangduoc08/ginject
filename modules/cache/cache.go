package cache

import (
	"context"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	SetNX(ctx context.Context, key string, val []byte, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
	Keys(ctx context.Context) []string
	TTL(ctx context.Context, key string) (time.Duration, bool)
}
