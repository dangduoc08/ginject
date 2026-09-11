package core

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/memorybroker"
)

func TestStress_WSConnmgr_ConcurrentLifecycle(t *testing.T) {
	const workers = 12
	const iterations = 60

	sharedConn, cleanup := wslTestConn(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	connmgr.maxConns = 0

	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				id := fmt.Sprintf("conn-%d-%d", w, i)

				c := connmgr.Register(id, sharedConn)
				if c == nil {
					continue
				}

				topic := fmt.Sprintf("topic.%d", i%4)
				_ = connmgr.Subscribe(id, topic, func(*memorybroker.Message) {})
				connmgr.isSubscribed(id, topic)
				connmgr.touch(id)
				connmgr.Count()
				connmgr.Get(id)
				_ = connmgr.Unsubscribe(id, topic)

				connmgr.Unregister(id)
				connmgr.Unregister(id)
			}
		}(w)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			connmgr.reapDeadConns(time.Nanosecond)
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()

	if got := connmgr.Count(); got != 0 {
		t.Errorf("every connection must be released after the churn, %d still tracked", got)
	}

	connmgr.mu.RLock()
	leftover := len(connmgr.subscriptions)
	connmgr.mu.RUnlock()
	if leftover != 0 {
		t.Errorf("every subscription list must be released after the churn, %d still tracked", leftover)
	}
}

func TestStress_WSConnmgr_MaxConnsEnforcedUnderConcurrency(t *testing.T) {
	sharedConn, cleanup := wslTestConn(t)
	defer cleanup()

	const limit = 10
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	connmgr.maxConns = limit

	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0

	for w := 0; w < 32; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			if c := connmgr.Register(fmt.Sprintf("c-%d", w), sharedConn); c != nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	if accepted != limit {
		t.Errorf("concurrent Register must admit exactly maxConns connections, admitted %d want %d", accepted, limit)
	}
	if got := connmgr.Count(); got != limit {
		t.Errorf("tracked connections must equal maxConns, got %d want %d", got, limit)
	}
}
