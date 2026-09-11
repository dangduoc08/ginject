package memorybroker

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
)

// ────────────────────────────────────────────────────────────────────────────
// Helpers
// ────────────────────────────────────────────────────────────────────────────

func newBroker(t *testing.T) Broker {
	t.Helper()
	b := NewMemoryBroker()
	t.Cleanup(func() { _ = b.Close() })
	return b
}

// ────────────────────────────────────────────────────────────────────────────
// Happy path
// ────────────────────────────────────────────────────────────────────────────

func TestSubscribeAndPublish(t *testing.T) {
	b := newBroker(t)

	var got *Message
	_, err := b.Subscribe("user.created", func(m *Message) { got = m })
	if err != nil {
		t.Fatal(err)
	}

	if err := b.Publish("user.created", "alice"); err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Error(test.DiffMessage(nil, "non-nil *Message", "handler should have been called"))
	} else {
		if got.Topic != "user.created" {
			t.Error(test.DiffMessage(got.Topic, "user.created", "message topic"))
		}
		if got.Payload != "alice" {
			t.Error(test.DiffMessage(got.Payload, "alice", "message payload"))
		}
		if got.Timestamp.IsZero() {
			t.Error(test.DiffMessage(got.Timestamp, "non-zero time", "message timestamp"))
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Wildcard *
// ────────────────────────────────────────────────────────────────────────────

func TestWildcardGlobal_ReceivesAllTopics(t *testing.T) {
	b := newBroker(t)

	var topics []string
	_, err := b.Subscribe("*", func(m *Message) { topics = append(topics, m.Topic) })
	if err != nil {
		t.Fatal(err)
	}

	_ = b.Publish("a", nil)
	_ = b.Publish("b.c", nil)
	_ = b.Publish("x.y.z", nil)

	if len(topics) != 3 {
		t.Error(test.DiffMessage(len(topics), 3, "global wildcard should receive every published message"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Wildcard prefix.*
// ────────────────────────────────────────────────────────────────────────────

func TestWildcardPrefix_ReceivesMatchingPrefix(t *testing.T) {
	b := newBroker(t)

	var received []string
	_, err := b.Subscribe("order.*", func(m *Message) { received = append(received, m.Topic) })
	if err != nil {
		t.Fatal(err)
	}

	_ = b.Publish("order.created", nil)
	_ = b.Publish("order.shipped", nil)
	_ = b.Publish("user.created", nil) // should NOT match

	if len(received) != 2 {
		t.Error(test.DiffMessage(len(received), 2, "prefix wildcard should match only 'order.*' topics"))
	}
	for _, top := range received {
		if len(top) < 6 || top[:6] != "order." {
			t.Error(test.DiffMessage(top, "order.*", "received topic should have 'order.' prefix"))
		}
	}
}

func TestWildcardPrefix_DoesNotMatchUnrelatedTopics(t *testing.T) {
	b := newBroker(t)

	var count int
	_, err := b.Subscribe("foo.*", func(_ *Message) { count++ })
	if err != nil {
		t.Fatal(err)
	}

	_ = b.Publish("bar.x", nil)
	_ = b.Publish("foobar.x", nil)
	_ = b.Publish("foo", nil) // no dot → no prefix match

	if count != 0 {
		t.Error(test.DiffMessage(count, 0, "prefix wildcard should not match unrelated topics"))
	}
}

// With the trailing "*" now greedy, two suffix-wildcard subscriptions at
// different depths ("chat.*" and "chat.room.*") can both match the same
// published topic. Publish must fan out to every matching prefix bucket,
// not just the deepest/most specific one.
func TestWildcardPrefix_FansOutAcrossOverlappingDepths(t *testing.T) {
	b := newBroker(t)

	var shallow, deep int
	_, _ = b.Subscribe("chat.*", func(_ *Message) { shallow++ })
	_, _ = b.Subscribe("chat.room.*", func(_ *Message) { deep++ })

	_ = b.Publish("chat.room.5", nil)

	if shallow != 1 {
		t.Error(test.DiffMessage(shallow, 1, "chat.* should also receive a publish to chat.room.5"))
	}
	if deep != 1 {
		t.Error(test.DiffMessage(deep, 1, "chat.room.* should receive a publish to chat.room.5"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Unsubscribe
// ────────────────────────────────────────────────────────────────────────────

func TestUnsubscribe_HandlerNotCalledAfter(t *testing.T) {
	b := newBroker(t)

	var count int
	sub, err := b.Subscribe("evt", func(_ *Message) { count++ })
	if err != nil {
		t.Fatal(err)
	}

	_ = b.Publish("evt", nil)
	if err := b.Unsubscribe(sub); err != nil {
		t.Fatal(err)
	}
	_ = b.Publish("evt", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "handler should fire once before unsubscribe, zero times after"))
	}
}

func TestUnsubscribeViaInterface(t *testing.T) {
	b := newBroker(t)

	var count int
	sub, _ := b.Subscribe("topic", func(_ *Message) { count++ })
	_ = b.Publish("topic", nil)
	_ = sub.Unsubscribe()
	_ = b.Publish("topic", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "Subscription.Unsubscribe() should prevent further delivery"))
	}
}

func TestUnsubscribe_Nil_NoError(t *testing.T) {
	b := newBroker(t)
	if err := b.Unsubscribe(nil); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe(nil) should be a no-op"))
	}
}

func TestUnsubscribe_Nil_AlwaysNoOp_RegardlessOfBrokerState(t *testing.T) {
	b := NewMemoryBroker()
	if err := b.Unsubscribe(nil); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe(nil) on an open broker should be nil"))
	}
	_ = b.Close()
	if err := b.Unsubscribe(nil); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe(nil) on a closed broker should still be nil, per Unsubscribe's own unconditional nil-safety rule"))
	}
}

type fakeSubscription struct{}

func (fakeSubscription) ID() string         { return "fake" }
func (fakeSubscription) Topic() string      { return "fake" }
func (fakeSubscription) Unsubscribe() error { return nil }

func TestUnsubscribe_ForeignType_NoError(t *testing.T) {
	b := newBroker(t)
	if err := b.Unsubscribe(fakeSubscription{}); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe with a foreign Subscription implementation should be a no-op"))
	}
}

func TestUnsubscribe_CrossBroker_RejectedAndDoesNotRemove(t *testing.T) {
	bA := newBroker(t)
	bB := newBroker(t)

	var count int
	subA, err := bA.Subscribe("shared.topic", func(_ *Message) { count++ })
	if err != nil {
		t.Fatal(err)
	}

	if err := bB.Unsubscribe(subA); err != ErrForeignSubscription {
		t.Error(test.DiffMessage(err, ErrForeignSubscription, "Unsubscribe with a subscription from a different broker should be rejected"))
	}

	_ = bA.Publish("shared.topic", nil)
	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "subA should remain active on its own broker after a rejected cross-broker Unsubscribe"))
	}
}

func TestUnsubscribe_ForeignType_AfterClose_ReturnsErrClosed(t *testing.T) {
	b := NewMemoryBroker()
	_ = b.Close()

	if err := b.Unsubscribe(fakeSubscription{}); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "closed check should take precedence over the foreign-type no-op"))
	}
}

func TestUnsubscribe_CrossBroker_AfterClose_ReturnsErrClosed(t *testing.T) {
	bA := newBroker(t)
	bB := NewMemoryBroker()

	subA, err := bA.Subscribe("shared.topic", func(_ *Message) {})
	if err != nil {
		t.Fatal(err)
	}
	_ = bB.Close()

	if err := bB.Unsubscribe(subA); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "closed check should take precedence over the cross-broker rejection"))
	}
}

func TestUnsubscribe_AfterClose_TakesPrecedenceOverValidSub(t *testing.T) {
	b := NewMemoryBroker()
	sub, err := b.Subscribe("t", func(_ *Message) {})
	if err != nil {
		t.Fatal(err)
	}
	_ = b.Close()

	if err := b.Unsubscribe(sub); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "Unsubscribe of a real subscription after Close should still return ErrClosed"))
	}
}

func TestUnsubscribe_ComplexTopic_DoubleCall_NoError(t *testing.T) {
	b := newBroker(t)
	sub, err := b.Subscribe("tenant.*.user.created", func(*Message) {})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Unsubscribe(sub); err != nil {
		t.Fatal(err)
	}
	if err := b.Unsubscribe(sub); err != nil {
		t.Error(test.DiffMessage(err, nil, "double Unsubscribe on an already-emptied complex bucket should be a no-op"))
	}
}

func TestUnsubscribe_ExactTopic_DoubleCall_NoError(t *testing.T) {
	b := newBroker(t)
	sub, err := b.Subscribe("evt.exact", func(*Message) {})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Unsubscribe(sub); err != nil {
		t.Fatal(err)
	}
	if err := b.Unsubscribe(sub); err != nil {
		t.Error(test.DiffMessage(err, nil, "double Unsubscribe on an already-emptied exact bucket should be a no-op"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Close
// ────────────────────────────────────────────────────────────────────────────

func TestClose_ReturnErrClosed(t *testing.T) {
	b := NewMemoryBroker()
	_ = b.Close()

	if err := b.Publish("t", nil); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "Publish after Close should return ErrClosed"))
	}
	if _, err := b.Subscribe("t", func(_ *Message) {}); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "Subscribe after Close should return ErrClosed"))
	}
	if err := b.Unsubscribe(nil); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe(nil) is an unconditional no-op, even after Close"))
	}
	if err := b.PublishAsync("t", nil); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "PublishAsync after Close should return ErrClosed"))
	}
}

func TestClose_Idempotent(t *testing.T) {
	b := NewMemoryBroker()
	if err := b.Close(); err != nil {
		t.Fatal(err)
	}
	if err := b.Close(); err != nil {
		t.Error(test.DiffMessage(err, nil, "calling Close twice should not error"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// PublishAsync: fire-and-forget delivery + Close drains in-flight publishes
// ────────────────────────────────────────────────────────────────────────────

func TestPublishAsync_DeliversAll(t *testing.T) {
	b := newBroker(t)

	var count atomic.Int64
	_, _ = b.Subscribe("wp.topic", func(*Message) { count.Add(1) })

	const msgs = 100
	for i := range msgs {
		if err := b.PublishAsync("wp.topic", i); err != nil {
			t.Fatalf("PublishAsync error: %v", err)
		}
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if count.Load() == msgs {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("got %d deliveries, want %d", count.Load(), msgs)
}

func TestPublishAsync_EmptyTopic_ReturnsError(t *testing.T) {
	b := newBroker(t)
	if err := b.PublishAsync("", nil); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "empty topic in PublishAsync"))
	}
}

func TestClose_DrainsInFlightAsyncPublishes(t *testing.T) {
	var count atomic.Int64
	b := NewMemoryBroker()

	_, _ = b.Subscribe("drain.topic", func(*Message) {
		time.Sleep(1 * time.Millisecond)
		count.Add(1)
	})

	const msgs = 20
	for i := range msgs {
		_ = b.PublishAsync("drain.topic", i)
	}

	// Close must block until all in-flight async publishes finish.
	_ = b.Close()

	if count.Load() != msgs {
		t.Errorf("after Close: delivered %d, want %d — in-flight publishes were not drained", count.Load(), msgs)
	}
}

// TestClose_FromWithinPublishHandler_DoesNotDeadlock verifies that Close() is
// safe to call synchronously from within a handler dispatched by the
// synchronous Publish() — Publish is not tracked by the WaitGroup, so
// Close()'s wg.Wait() has nothing of this call's to wait on, and the
// snapshot-then-unlock-then-execute design means no lock is held during
// handler execution either.
func TestClose_FromWithinPublishHandler_DoesNotDeadlock(t *testing.T) {
	b := NewMemoryBroker()
	done := make(chan struct{})
	_, err := b.Subscribe("foo", func(*Message) {
		_ = b.Close()
		close(done)
	})
	if err != nil {
		t.Fatal(err)
	}

	go func() { _ = b.Publish("foo", nil) }()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() called from within a synchronous Publish handler deadlocked unexpectedly")
	}
}

// TestClose_FromWithinPublishAsyncHandler_Deadlocks pins a verified, structural
// limitation: PublishAsync's goroutine calls wg.Done() only after the handler
// returns, but Close() called from inside that same handler blocks in
// wg.Wait() until wg.Done() runs — the goroutine is waiting on its own
// completion. No implementation of "Close waits for every accepted
// PublishAsync" can support this without either deadlocking or silently
// excluding the caller's own in-flight publish (a weaker guarantee), so this
// call pattern is unsupported by contract rather than special-cased in code.
// If this test ever observes <-done instead of timing out, the limitation
// has been fixed by design — update this test and the README together.
func TestClose_FromWithinPublishAsyncHandler_Deadlocks(t *testing.T) {
	b := NewMemoryBroker()
	done := make(chan struct{})
	_, err := b.Subscribe("foo", func(*Message) {
		_ = b.Close()
		close(done)
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.PublishAsync("foo", nil); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
		t.Fatal("expected Close() called from inside a PublishAsync handler to deadlock via wg self-wait, but it returned")
	case <-time.After(300 * time.Millisecond):
	}
}

// TestClose_DoesNotWaitForConcurrentSyncPublish documents that Close() is
// only synchronized with in-flight PublishAsync goroutines (via wg) and with
// its own map mutation (via rwMu) — it does not wait for a synchronous
// Publish() call running on another goroutine to finish executing handlers.
func TestClose_DoesNotWaitForConcurrentSyncPublish(t *testing.T) {
	b := NewMemoryBroker()
	handlerStarted := make(chan struct{})
	handlerMayFinish := make(chan struct{})
	_, err := b.Subscribe("slow", func(*Message) {
		close(handlerStarted)
		<-handlerMayFinish
	})
	if err != nil {
		t.Fatal(err)
	}

	go func() { _ = b.Publish("slow", nil) }()
	<-handlerStarted

	closeDone := make(chan struct{})
	go func() {
		_ = b.Close()
		close(closeDone)
	}()

	select {
	case <-closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() unexpectedly waited for an in-flight synchronous Publish to finish")
	}
	close(handlerMayFinish)
}

func TestPublishAsync_RaceWithClose(t *testing.T) {
	for i := 0; i < 2000; i++ {
		b := NewMemoryBroker()
		_, _ = b.Subscribe("t", func(*Message) {})

		var wg sync.WaitGroup
		for j := 0; j < 8; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = b.PublishAsync("t", nil)
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Close()
		}()
		wg.Wait()
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Validation errors
// ────────────────────────────────────────────────────────────────────────────

func TestNilHandler_ReturnsError(t *testing.T) {
	b := newBroker(t)

	_, err := b.Subscribe("t", nil)
	if err != ErrNilHandler {
		t.Error(test.DiffMessage(err, ErrNilHandler, "nil handler should return ErrNilHandler"))
	}
}

func TestEmptyTopic_ReturnsError(t *testing.T) {
	b := newBroker(t)

	noop := func(_ *Message) {}
	if _, err := b.Subscribe("", noop); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "empty topic in Subscribe"))
	}
	if err := b.Publish("", nil); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "empty topic in Publish"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Panic recovery
// ────────────────────────────────────────────────────────────────────────────

func TestPanicRecovery_OtherHandlersStillReceive(t *testing.T) {
	b := newBroker(t)

	var after atomic.Int64
	_, _ = b.Subscribe("panic.topic", func(*Message) { panic("deliberate") })
	_, _ = b.Subscribe("panic.topic", func(*Message) { after.Add(1) })
	_, _ = b.Subscribe("panic.topic", func(*Message) { after.Add(1) })

	_ = b.Publish("panic.topic", nil)

	if n := after.Load(); n != 2 {
		t.Errorf("handlers after the panicking one: got %d, want 2", n)
	}
}

func TestPanicRecovery_PublishAsync_OtherHandlersStillReceive(t *testing.T) {
	b := newBroker(t)

	var after atomic.Int64
	_, _ = b.Subscribe("panic.async.topic", func(*Message) { panic("deliberate") })
	_, _ = b.Subscribe("panic.async.topic", func(*Message) { after.Add(1) })
	_, _ = b.Subscribe("panic.async.topic", func(*Message) { after.Add(1) })

	if err := b.PublishAsync("panic.async.topic", nil); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if after.Load() == 2 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Errorf("handlers after the panicking one: got %d, want 2 — a panic in one PublishAsync handler must not stop the others or crash the goroutine", after.Load())
}

// ────────────────────────────────────────────────────────────────────────────
// Empty bucket cleanup
// ────────────────────────────────────────────────────────────────────────────

func TestEmptyBucketCleanup_Exact(t *testing.T) {
	// Use the internal broker type to inspect the maps directly.
	b := NewMemoryBroker().(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	sub, _ := b.Subscribe("ephemeral.topic", func(*Message) {})
	_ = b.Unsubscribe(sub)

	b.rwMu.RLock()
	_, hasBucket := b.exactByTopic["ephemeral.topic"]
	b.rwMu.RUnlock()
	if hasBucket {
		t.Error("empty exact bucket should have been deleted after unsubscribe")
	}
}

func TestEmptyBucketCleanup_Prefix(t *testing.T) {
	b := NewMemoryBroker().(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	sub, _ := b.Subscribe("order.*", func(*Message) {})
	_ = b.Unsubscribe(sub)

	b.rwMu.RLock()
	_, hasBucket := b.prefixByPrefix["order"]
	b.rwMu.RUnlock()
	if hasBucket {
		t.Error("empty prefix bucket should have been deleted after unsubscribe")
	}
}

func TestEmptyBucketCleanup_Complex(t *testing.T) {
	b := NewMemoryBroker().(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	sub, _ := b.Subscribe("tenant.*.user.created", func(*Message) {})
	_ = b.Unsubscribe(sub)

	b.rwMu.RLock()
	_, hasBucket := b.complexByTopic["tenant.*.user.created"]
	b.rwMu.RUnlock()
	if hasBucket {
		t.Error("empty complex bucket should have been deleted after unsubscribe")
	}
}

func TestEmptyBucketCleanup_Global(t *testing.T) {
	b := NewMemoryBroker().(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	sub, _ := b.Subscribe("*", func(*Message) {})
	_ = b.Unsubscribe(sub)

	b.rwMu.RLock()
	_, stillPresent := b.globalByID[sub.ID()]
	b.rwMu.RUnlock()
	if stillPresent {
		t.Error("global subscription should have been removed from globalByID after unsubscribe")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Concurrent safety (run with -race)
// ────────────────────────────────────────────────────────────────────────────

func TestConcurrentPublish(t *testing.T) {
	b := newBroker(t)

	var received atomic.Int64
	_, _ = b.Subscribe("concurrent", func(_ *Message) { received.Add(1) })
	_, _ = b.Subscribe("*", func(_ *Message) {}) // extra subscriber

	const goroutines = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = b.Publish("concurrent", nil)
		}()
	}
	wg.Wait()

	if got := received.Load(); got != goroutines {
		t.Error(test.DiffMessage(got, goroutines, "each goroutine publishes one message; exact count expected"))
	}
}

func TestConcurrentSubscribeUnsubscribe(t *testing.T) {
	b := newBroker(t)

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			sub, err := b.Subscribe("race.topic", func(_ *Message) {})
			if err == nil {
				_ = b.Unsubscribe(sub)
			}
		}()
		go func() {
			defer wg.Done()
			_ = b.Publish("race.topic", nil)
		}()
	}
	wg.Wait()
	// If we get here without the race detector firing, we're good.
}

// TestConcurrentPublishSubscribeUnsubscribeClose stresses Publish, Subscribe,
// Unsubscribe, and Close all racing against each other on the same broker —
// the scenario Close's mu-based synchronization with the rest of the API
// exists to make safe.
func TestConcurrentPublishSubscribeUnsubscribeClose(t *testing.T) {
	for attempt := 0; attempt < 20; attempt++ {
		b := NewMemoryBroker()

		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					_ = b.Publish("stress", nil)
				}
			}()
		}
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					sub, err := b.Subscribe("stress", func(_ *Message) {})
					if err == nil {
						_ = b.Unsubscribe(sub)
					}
				}
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Close()
		}()
		wg.Wait()
	}
}

// TestConcurrentSubscribe_NeverLeaksAfterClose guards against a Subscribe/Close
// TOCTOU race: Subscribe must not be able to insert a subscription into the
// broker's maps after Close has committed to closing, even when the two race.
// Once Close returns, no subscription bucket should remain.
func TestConcurrentSubscribe_NeverLeaksAfterClose(t *testing.T) {
	for attempt := 0; attempt < 300; attempt++ {
		b := NewMemoryBroker().(*MemoryBroker)

		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = b.Subscribe("race.sub", func(_ *Message) {})
			}()
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Close()
		}()
		wg.Wait()

		b.rwMu.RLock()
		leaked := len(b.exactByTopic) + len(b.prefixByPrefix) + len(b.globalByID) + len(b.complexByTopic)
		b.rwMu.RUnlock()
		if leaked != 0 {
			t.Fatalf("attempt %d: %d subscription bucket(s) leaked after Close returned", attempt, leaked)
		}
	}
}

// TestHandlerReentrancy_SubscribeUnsubscribePublishPublishAsync_NoDeadlock
// verifies a handler can call back into every broker method — except Close,
// which has its own documented limitation — without deadlocking, confirming
// no lock is held across handler execution.
func TestHandlerReentrancy_SubscribeUnsubscribePublishPublishAsync_NoDeadlock(t *testing.T) {
	b := newBroker(t)

	var innerPublishCount atomic.Int64
	done := make(chan struct{})

	sub, err := b.Subscribe("reentrant.trigger", func(*Message) {
		newSub, err := b.Subscribe("reentrant.inner", func(*Message) { innerPublishCount.Add(1) })
		if err != nil {
			t.Error(err)
		}
		if err := b.Publish("reentrant.inner", nil); err != nil {
			t.Error(err)
		}
		if err := b.PublishAsync("reentrant.inner", nil); err != nil {
			t.Error(err)
		}
		if err := b.Unsubscribe(newSub); err != nil {
			t.Error(err)
		}
		close(done)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = b.Unsubscribe(sub) }()

	if err := b.Publish("reentrant.trigger", nil); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handler reentering Subscribe/Publish/PublishAsync/Unsubscribe deadlocked")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Subscription fields
// ────────────────────────────────────────────────────────────────────────────

func TestSubscription_IDAndTopic(t *testing.T) {
	b := newBroker(t)

	sub, err := b.Subscribe("my.topic", func(_ *Message) {})
	if err != nil {
		t.Fatal(err)
	}

	if sub.ID() == "" {
		t.Error(test.DiffMessage("", "non-empty UUID", "Subscription.ID() should be non-empty"))
	}
	if sub.Topic() != "my.topic" {
		t.Error(test.DiffMessage(sub.Topic(), "my.topic", "Subscription.Topic() mismatch"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Multiple subscribers receive the same message
// ────────────────────────────────────────────────────────────────────────────

func TestMultipleSubscribers_SameMessage(t *testing.T) {
	b := newBroker(t)

	var msgs [3]*Message
	_, _ = b.Subscribe("e", func(m *Message) { msgs[0] = m })
	_, _ = b.Subscribe("e", func(m *Message) { msgs[1] = m })
	_, _ = b.Subscribe("e", func(m *Message) { msgs[2] = m })

	_ = b.Publish("e", "payload")

	for i, m := range msgs {
		if m == nil {
			t.Errorf("subscriber %d: %s", i, test.DiffMessage(nil, "non-nil *Message", "should have been called"))
			continue
		}
		if m.Payload != "payload" {
			t.Errorf("subscriber %d: %s", i, test.DiffMessage(m.Payload, "payload", "received wrong payload"))
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Topic pattern kinds
// ────────────────────────────────────────────────────────────────────────────

func TestSubscribe_SuffixWildcard_MatchesDeepTopics(t *testing.T) {
	b := newBroker(t)
	var received []string
	_, _ = b.Subscribe("user.*", func(m *Message) { received = append(received, m.Topic) })

	_ = b.Publish("user.created", nil)
	_ = b.Publish("user.profile.updated", nil)
	_ = b.Publish("user.profile.avatar.changed", nil)
	_ = b.Publish("order.created", nil)

	if len(received) != 3 {
		t.Error(test.DiffMessage(len(received), 3, "user.* should now greedily match 3 deep topics"))
	}
}

func TestSubscribe_SuffixWildcard_DoesNotMatchParent(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("user.*", func(*Message) { count++ })

	_ = b.Publish("user", nil)

	if count != 0 {
		t.Error(test.DiffMessage(count, 0, "user.* must not match 'user' itself"))
	}
}

func TestSubscribe_SuffixWildcard_MatchesMultipleLevels(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("user.*", func(*Message) { count++ })

	_ = b.Publish("user.created", nil)
	_ = b.Publish("user.profile.updated", nil)

	if count != 2 {
		t.Error(test.DiffMessage(count, 2, "user.* should now match any depth beyond the prefix, not just one level"))
	}
}

func TestSubscribe_Global_MatchesAll(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("*", func(*Message) { count++ })

	_ = b.Publish("a", nil)
	_ = b.Publish("a.b", nil)
	_ = b.Publish("a.b.c", nil)

	if count != 3 {
		t.Error(test.DiffMessage(count, 3, "* alone should match every published topic"))
	}
}

func TestSubscribe_BackwardCompat_StarAlias(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("*", func(*Message) { count++ })

	_ = b.Publish("a.b.c", nil)
	_ = b.Publish("x", nil)

	if count != 2 {
		t.Error(test.DiffMessage(count, 2, "* alone should match every topic regardless of depth"))
	}
}

func TestSubscribe_Complex_MiddleWildcard(t *testing.T) {
	b := newBroker(t)
	var received []string
	_, _ = b.Subscribe("tenant.*.user.created", func(m *Message) { received = append(received, m.Topic) })

	_ = b.Publish("tenant.1.user.created", nil)
	_ = b.Publish("tenant.abc.user.created", nil)
	_ = b.Publish("tenant.1.user.updated", nil)
	_ = b.Publish("tenant.1.2.user.created", nil)

	if len(received) != 2 {
		t.Error(test.DiffMessage(len(received), 2, "tenant.*.user.created should match 2 topics"))
	}
}

func TestSubscribe_Complex_MiddleAndTrailingWildcard(t *testing.T) {
	b := newBroker(t)
	var received []string
	_, _ = b.Subscribe("tenant.*.user.*", func(m *Message) { received = append(received, m.Topic) })

	_ = b.Publish("tenant.1.user.created", nil)
	_ = b.Publish("tenant.abc.user.profile.updated", nil)
	_ = b.Publish("tenant.1.admin.created", nil)
	_ = b.Publish("tenant.1.user", nil)

	if len(received) != 2 {
		t.Error(test.DiffMessage(len(received), 2, "tenant.*.user.* should match 2 topics"))
	}
}

func TestCallHandler_PanickingSubscriber_IsIsolatedAndReported(t *testing.T) {
	var mu sync.Mutex
	var gotTopic string
	var gotRecovered any

	b := NewMemoryBroker(WithPanicHandler(func(topic string, recovered any) {
		mu.Lock()
		gotTopic = topic
		gotRecovered = recovered
		mu.Unlock()
	}))
	defer func() { _ = b.Close() }()

	var survivorCalls int
	if _, err := b.Subscribe("boom", func(*Message) {
		panic("subscriber exploded")
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Subscribe("boom", func(*Message) {
		survivorCalls++
	}); err != nil {
		t.Fatal(err)
	}

	if err := b.Publish("boom", nil); err != nil {
		t.Fatal(err)
	}

	if survivorCalls != 1 {
		t.Error(test.DiffMessage(survivorCalls, 1, "a panicking subscriber must not stop the other subscribers"))
	}

	mu.Lock()
	defer mu.Unlock()
	if gotTopic != "boom" {
		t.Error(test.DiffMessage(gotTopic, "boom", "the panic report must name the topic"))
	}
	if gotRecovered != "subscriber exploded" {
		t.Error(test.DiffMessage(gotRecovered, "subscriber exploded", "the panic report must carry the recovered value"))
	}
}

func TestCallHandler_PanicHandlerThatPanics_DoesNotEscape(t *testing.T) {
	b := NewMemoryBroker(WithPanicHandler(func(string, any) {
		panic("reporter exploded too")
	}))
	defer func() { _ = b.Close() }()

	if _, err := b.Subscribe("boom", func(*Message) { panic("subscriber exploded") }); err != nil {
		t.Fatal(err)
	}

	if err := b.Publish("boom", nil); err != nil {
		t.Fatal(err)
	}
}

func TestNewMemoryBroker_NilOption_Ignored(t *testing.T) {
	b := NewMemoryBroker(nil)
	defer func() { _ = b.Close() }()

	if err := b.Publish("t", nil); err != nil {
		t.Error(err)
	}
}

func TestPublish_SuffixWildcardMatchesEveryDotPrefix(t *testing.T) {
	cases := []struct {
		pattern string
		topic   string
		want    bool
	}{
		{"a.*", "a.b.c.d", true},
		{"a.b.*", "a.b.c.d", true},
		{"a.b.c.*", "a.b.c.d", true},
		{"a.b.c.d.*", "a.b.c.d", false},
		{"a.*", "a", false},
		{"a.*", "ax.b", false},
		{"trailing.*", "trailing.", true},
		{"a.*", "a..b", true},
	}

	for _, c := range cases {
		b := NewMemoryBroker()

		var fired atomic.Int64
		if _, err := b.Subscribe(c.pattern, func(*Message) { fired.Add(1) }); err != nil {
			t.Fatal(err)
		}
		if err := b.Publish(c.topic, nil); err != nil {
			t.Fatal(err)
		}
		_ = b.Close()

		got := fired.Load() == 1
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "pattern "+c.pattern+" against topic "+c.topic))
		}
	}
}
