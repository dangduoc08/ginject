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
	Set(string, T, time.Duration)
	Del(string) bool
	Has(string) bool
	Clear() bool
}

type CacheOpts struct {
	Strategy uint16
	Cap      int
}

func New[T any](opts CacheOpts) CacheModuler[T] {
	switch opts.Strategy {
	case LFU:
		return &lfu[T]{
			wg:        &sync.WaitGroup{},
			values:    &sync.Map{},
			freqMap:   &sync.Map{},
			cap:       opts.Cap,
			leastFreq: 0,
		}
	}
	panic(fmt.Errorf("%v is unavailable strategy", opts.Strategy))
}
