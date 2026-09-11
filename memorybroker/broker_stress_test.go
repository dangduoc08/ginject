package memorybroker

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestStress_ConcurrentSubscribeUnsubscribePublishClose(t *testing.T) {
	const goroutines = 16
	const iterations = 200

	b := NewMemoryBroker()

	var delivered atomic.Int64
	var wg sync.WaitGroup

	topics := []string{"a.b.c", "a.b.d", "a.*", "a.>", ">", "x.y"}

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				topic := topics[(g+i)%len(topics)]

				sub, err := b.Subscribe(topic, func(m *Message) {
					delivered.Add(1)
				})
				if err != nil {
					continue
				}

				_ = b.Publish("a.b.c", i)
				_ = b.PublishAsync("a.b.d", i)

				if err := b.Unsubscribe(sub); err != nil {
					continue
				}
			}
		}(g)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = b.Publish("x.y", i)
			_ = b.PublishAsync("a.b.c", i)
		}
	}()

	wg.Wait()

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Error(err)
	}

	if _, err := b.Subscribe("a.b.c", func(*Message) {}); err != ErrClosed {
		t.Errorf("Subscribe after Close must return ErrClosed, got %v", err)
	}
	if err := b.Publish("a.b.c", 1); err != ErrClosed {
		t.Errorf("Publish after Close must return ErrClosed, got %v", err)
	}
	if err := b.PublishAsync("a.b.c", 1); err != ErrClosed {
		t.Errorf("PublishAsync after Close must return ErrClosed, got %v", err)
	}
}

func TestStress_PublishAsyncRacingClose_NoPanicNoLostWaitGroup(t *testing.T) {
	for round := 0; round < 50; round++ {
		b := NewMemoryBroker()
		if _, err := b.Subscribe("t", func(*Message) {}); err != nil {
			t.Fatal(err)
		}

		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 40; j++ {
					_ = b.PublishAsync("t", j)
				}
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Close()
		}()

		wg.Wait()
		_ = b.Close()
	}
}

func TestStress_HandlerReentrantPublishAndUnsubscribe(t *testing.T) {
	b := NewMemoryBroker()
	defer func() { _ = b.Close() }()

	var inner atomic.Int64

	if _, err := b.Subscribe("inner", func(*Message) {
		inner.Add(1)
	}); err != nil {
		t.Fatal(err)
	}

	var self Subscription
	sub, err := b.Subscribe("outer", func(*Message) {
		_ = b.Publish("inner", nil)
		_ = b.Unsubscribe(self)
	})
	if err != nil {
		t.Fatal(err)
	}
	self = sub

	done := make(chan struct{})
	go func() {
		_ = b.Publish("outer", nil)
		close(done)
	}()

	select {
	case <-done:
	case <-make(chan struct{}):
	}

	if inner.Load() != 1 {
		t.Errorf("re-entrant Publish from a handler must run, got %d deliveries", inner.Load())
	}
}
