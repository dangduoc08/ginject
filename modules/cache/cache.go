package cache

import (
	"context"
	"errors"
	"time"
)

var ErrEmptyKey = errors.New("cache: key must not be empty")

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	SetNX(ctx context.Context, key string, val []byte, ttl time.Duration) (bool, error)
	Delete(ctx context.Context, key string) error
	Keys(ctx context.Context) []string
	TTL(ctx context.Context, key string) (time.Duration, bool)
}
