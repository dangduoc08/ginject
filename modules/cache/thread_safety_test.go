package cache

import (
	"sync"
	"testing"
)

func TestUpsertDLLRaceCondition(t *testing.T) {
	cacheModule := New[string](CacheOpts{
		Strategy: LFU,
		Cap:      100,
	})

	var expect1 = 0
	wg := &sync.WaitGroup{}

	for i := 0; i < 5000; i++ {
		expect1 += i
		wg.Add(1)

		go func(i int64, wg *sync.WaitGroup) {
			cacheModule.upsertDLLByFreq(i)
			wg.Done()
		}(int64(i), wg)
	}

	wg.Wait()

	var output1 int64 = 0
	cacheModule.freqMap.Range(func(key, value any) bool {
		output1 += key.(int64)
		return true
	})

	if int64(expect1) != output1 {
		t.Errorf("upsertDLLByFreq caused race condition. Total created key expect = %v; output = %v", expect1, output1)
	}
}
