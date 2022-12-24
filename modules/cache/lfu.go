package cache

import (
	"sync"
	"sync/atomic"
	"time"
)

type lfuValues[U any, T object[U]] struct {
	node *node[T]
	freq int64
	ex   time.Duration
}

type lfu[U any, T object[U]] struct {
	mu              *sync.Mutex
	values, freqMap *sync.Map // use lfuValues
	cap, leastF     int64
}

func (c *lfu[U, T]) getDLLByFreq(f int64) *DLL[object[U]] {
	dll, ok := c.freqMap.Load(f)
	if !ok {
		return nil
	}
	return dll.(*DLL[object[U]])
}

func (c *lfu[U, T]) createDLLByFreq(f int64) *DLL[object[U]] {
	dll := NewDLL[object[U]]()
	c.freqMap.Store(f, dll)
	return dll
}

// thread safety
// cmd: go test --run UpsertDLLRaceCondition
func (c *lfu[U, T]) upsertDLLByFreq(f int64) *DLL[object[U]] {
	dll := c.getDLLByFreq(f)
	if dll == nil {
		dll = c.createDLLByFreq(f)
	}
	return dll
}

func (c *lfu[U, T]) Get(k string) (U, bool) {
	resp, ok := c.values.Load(k)
	if !ok {
		var zero U
		return zero, false
	}

	lfuValue := resp.(*lfuValues[U, object[U]])
	if isExpired(lfuValue.ex) {
		c.values.Delete(k)
		atomic.AddInt64(&c.cap, 1)

		ll := c.getDLLByFreq(lfuValue.freq)
		if ll != nil {
			ll.delete(lfuValue.node)

			if ll.size == 0 {

				// delete DLL by freq map
				c.freqMap.Delete(lfuValue.freq)

				var leastF int64 = -1
				c.freqMap.Range(func(f, value any) bool {
					if leastF == -1 || leastF < f.(int64) {
						leastF = f.(int64)
					}
					return true
				})
				c.leastF = leastF
			}
		}

		var zero U
		return zero, false
	}

	ll := c.getDLLByFreq(lfuValue.freq)
	if ll != nil {
		ll.delete(lfuValue.node)
		if ll.size == 0 {

			// delete DLL by freq map
			c.freqMap.Delete(lfuValue.freq)

			if c.leastF == lfuValue.freq {
				atomic.AddInt64(&c.leastF, 1)
			}
		}
	}
	atomic.AddInt64(&lfuValue.freq, 1)
	newLL := c.upsertDLLByFreq(lfuValue.freq)
	newNode := newLL.unshift(lfuValue.node.data)
	c.values.Store(k, &lfuValues[U, object[U]]{
		freq: lfuValue.freq,
		ex:   lfuValue.ex,
		node: newNode,
	})

	return lfuValue.node.data.value, true
}

func (c *lfu[U, T]) Set(k string, d U, ex time.Duration) {
	if ex == 0 {
		return
	}

	var initFreq int64 = 0

	if c.cap <= 0 {
		ll := c.getDLLByFreq(c.leastF)
		if ll != nil {
			rmNode := ll.pop()
			var d = rmNode.data

			// delete value from cache
			c.values.Delete(d.key)
			if ll.size == 0 {

				// delete DLL by freq map
				c.freqMap.Delete(c.leastF)
			}
		}

		// increase cap
		atomic.AddInt64(&c.cap, 1)
	}

	c.leastF = initFreq
	ll := c.upsertDLLByFreq(initFreq)
	obj := object[U]{
		key:   k,
		value: d,
	}
	n := ll.unshift(obj)

	c.values.Store(k, &lfuValues[U, object[U]]{
		freq: initFreq,
		ex:   setExpiry(ex),
		node: n,
	})
	atomic.AddInt64(&c.cap, -1)
}

func (c *lfu[U, T]) Del(k string) bool {
	return true
}

func (c *lfu[U, T]) Has(k string) bool {
	return true
}

func (c *lfu[U, T]) Clear() bool {
	return true
}
