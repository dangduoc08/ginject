package cache

import (
	"fmt"
	"sync"
	"time"
)

const (
	LFU = iota + 1
)

type CacheModuler[T any] interface {
	Get(string) (T, bool)
	Set(string, T, time.Duration) // ex in milliseconds
	Del(string) bool
	Has(string) bool
	Clear() bool
}

type CacheOpts struct {
	Strategy uint16
	Cap      int64
}

type object[U any] struct {
	key   string
	value U
}

func New[U any](opts CacheOpts) *lfu[U, object[U]] {
	switch opts.Strategy {
	case LFU:
		return &lfu[U, object[U]]{
			mu:      &sync.Mutex{},
			values:  &sync.Map{},
			freqMap: &sync.Map{},
			cap:     opts.Cap,
			leastF:  0,
		}
	}
	panic(fmt.Errorf("%v is unavailable strategy", opts.Strategy))
}
