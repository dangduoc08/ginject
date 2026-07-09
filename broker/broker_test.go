package broker

import (
	"sort"
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
	b := New()
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
		if got.ID == "" {
			t.Error(test.DiffMessage("", "non-empty UUID", "message ID"))
		}
		if got.Timestamp.IsZero() {
			t.Error(test.DiffMessage(got.Timestamp, "non-zero time", "message timestamp"))
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Once
// ────────────────────────────────────────────────────────────────────────────

func TestOnce_FiresExactlyOnce(t *testing.T) {
	b := newBroker(t)

	var count int
	_, err := b.Once("ping", func(_ *Message) { count++ })
	if err != nil {
		t.Fatal(err)
	}

	_ = b.Publish("ping", nil)
	_ = b.Publish("ping", nil)
	_ = b.Publish("ping", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "Once handler should fire exactly once"))
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

// ────────────────────────────────────────────────────────────────────────────
// Off
// ────────────────────────────────────────────────────────────────────────────

func TestOff_RemovesAllHandlersForTopic(t *testing.T) {
	b := newBroker(t)

	var count int
	inc := func(_ *Message) { count++ }
	_, _ = b.Subscribe("click", inc)
	_, _ = b.Subscribe("click", inc)
	_, _ = b.Subscribe("click", inc)

	_ = b.Publish("click", nil) // count = 3

	if err := b.Off("click"); err != nil {
		t.Fatal(err)
	}
	_ = b.Publish("click", nil) // should add 0

	if count != 3 {
		t.Error(test.DiffMessage(count, 3, "Off should remove all handlers; only 3 deliveries expected"))
	}
}

func TestOff_WildcardGlobal(t *testing.T) {
	b := newBroker(t)

	var count int
	_, _ = b.Subscribe("*", func(_ *Message) { count++ })
	_ = b.Publish("x", nil)
	_ = b.Off("*")
	_ = b.Publish("x", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "Off('*') should remove global handler"))
	}
}

func TestOff_WildcardPrefix(t *testing.T) {
	b := newBroker(t)

	var count int
	_, _ = b.Subscribe("ns.*", func(_ *Message) { count++ })
	_ = b.Publish("ns.a", nil)
	_ = b.Off("ns.*")
	_ = b.Publish("ns.b", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "Off('ns.*') should remove prefix handler"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ListenerCount
// ────────────────────────────────────────────────────────────────────────────

func TestListenerCount(t *testing.T) {
	b := newBroker(t)

	noop := func(_ *Message) {}
	_, _ = b.Subscribe("t", noop)
	_, _ = b.Subscribe("t", noop)
	sub3, _ := b.Subscribe("t", noop)

	if n := b.ListenerCount("t"); n != 3 {
		t.Error(test.DiffMessage(n, 3, "ListenerCount should be 3"))
	}

	_ = b.Unsubscribe(sub3)
	if n := b.ListenerCount("t"); n != 2 {
		t.Error(test.DiffMessage(n, 2, "ListenerCount should be 2 after unsubscribe"))
	}
}

func TestListenerCount_Wildcard(t *testing.T) {
	b := newBroker(t)

	noop := func(_ *Message) {}
	_, _ = b.Subscribe("*", noop)
	_, _ = b.Subscribe("*", noop)
	_, _ = b.Subscribe("a.*", noop)

	if n := b.ListenerCount("*"); n != 2 {
		t.Error(test.DiffMessage(n, 2, "ListenerCount('*') should be 2"))
	}
	if n := b.ListenerCount("a.*"); n != 1 {
		t.Error(test.DiffMessage(n, 1, "ListenerCount('a.*') should be 1"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Topics
// ────────────────────────────────────────────────────────────────────────────

func TestTopics(t *testing.T) {
	b := newBroker(t)

	noop := func(_ *Message) {}
	_, _ = b.Subscribe("alpha", noop)
	_, _ = b.Subscribe("beta.*", noop)
	_, _ = b.Subscribe("*", noop)

	topics := b.Topics()
	sort.Strings(topics)

	want := []string{"*", "alpha", "beta.*"}
	if len(topics) != len(want) {
		t.Fatal(test.DiffMessage(topics, want, "Topics() length mismatch"))
	}
	for i, w := range want {
		if topics[i] != w {
			t.Error(test.DiffMessage(topics[i], w, "Topics() element mismatch"))
		}
	}
}

func TestTopics_EmptyAfterClear(t *testing.T) {
	b := newBroker(t)

	_, _ = b.Subscribe("foo", func(_ *Message) {})
	_ = b.Clear()

	topics := b.Topics()
	if len(topics) != 0 {
		t.Error(test.DiffMessage(len(topics), 0, "Topics() should be empty after Clear"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Clear
// ────────────────────────────────────────────────────────────────────────────

func TestClear(t *testing.T) {
	b := newBroker(t)

	var count int
	_, _ = b.Subscribe("e1", func(_ *Message) { count++ })
	_, _ = b.Subscribe("e2", func(_ *Message) { count++ })
	_, _ = b.Subscribe("*", func(_ *Message) { count++ })

	_ = b.Clear()
	_ = b.Publish("e1", nil)
	_ = b.Publish("e2", nil)

	if count != 0 {
		t.Error(test.DiffMessage(count, 0, "Clear should remove all subscriptions"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Close
// ────────────────────────────────────────────────────────────────────────────

func TestClose_ReturnErrClosed(t *testing.T) {
	b := New()
	_ = b.Close()

	if err := b.Publish("t", nil); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "Publish after Close should return ErrClosed"))
	}
	if _, err := b.Subscribe("t", func(_ *Message) {}); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "Subscribe after Close should return ErrClosed"))
	}
	if err := b.PublishAsync("t", nil); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "PublishAsync after Close should return ErrClosed"))
	}
	if err := b.Off("t"); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "Off after Close should return ErrClosed"))
	}
	if err := b.Clear(); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "Clear after Close should return ErrClosed"))
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

	_, err = b.Once("t", nil)
	if err != ErrNilHandler {
		t.Error(test.DiffMessage(err, ErrNilHandler, "Once with nil handler should return ErrNilHandler"))
	}
}

func TestEmptyTopic_ReturnsError(t *testing.T) {
	b := newBroker(t)

	noop := func(_ *Message) {}
	if _, err := b.Subscribe("", noop); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "empty topic in Subscribe"))
	}
	if _, err := b.Once("", noop); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "empty topic in Once"))
	}
	if err := b.Publish("", nil); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "empty topic in Publish"))
	}
	if err := b.PublishAsync("", nil); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "empty topic in PublishAsync"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// PublishAsync
// ────────────────────────────────────────────────────────────────────────────

func TestPublishAsync_HandlerEventuallyFires(t *testing.T) {
	b := newBroker(t)

	var count atomic.Int32
	_, _ = b.Subscribe("async.topic", func(_ *Message) { count.Add(1) })

	for i := 0; i < 5; i++ {
		if err := b.PublishAsync("async.topic", i); err != nil {
			t.Fatal(err)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if count.Load() == 5 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error(test.DiffMessage(count.Load(), int32(5), "PublishAsync: all 5 messages should be delivered"))
}

// ────────────────────────────────────────────────────────────────────────────
// Once with wildcard
// ────────────────────────────────────────────────────────────────────────────

func TestOnce_WithGlobalWildcard(t *testing.T) {
	b := newBroker(t)

	var count int
	_, err := b.Once("*", func(_ *Message) { count++ })
	if err != nil {
		t.Fatal(err)
	}

	_ = b.Publish("a", nil)
	_ = b.Publish("b", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "Once('*') should fire exactly once across all topics"))
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

func TestOnce_ConcurrentPublish_FiresExactlyOnce(t *testing.T) {
	for attempt := 0; attempt < 200; attempt++ {
		b := New()
		var count atomic.Int64

		_, _ = b.Once("ev", func(*Message) { count.Add(1) })

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = b.Publish("ev", nil)
			}()
		}
		wg.Wait()
		_ = b.Close()

		if n := count.Load(); n != 1 {
			t.Errorf("attempt %d: once handler fired %d times, want exactly 1", attempt, n)
			return
		}
	}
}

func TestOnce_ConcurrentPublish_PrefixWildcard_FiresExactlyOnce(t *testing.T) {
	for attempt := 0; attempt < 200; attempt++ {
		b := New()
		var count atomic.Int64

		_, _ = b.Once("user.*", func(*Message) { count.Add(1) })

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = b.Publish("user.created", nil)
			}()
		}
		wg.Wait()
		_ = b.Close()

		if n := count.Load(); n != 1 {
			t.Errorf("attempt %d: once handler fired %d times, want exactly 1", attempt, n)
			return
		}
	}
}

func TestOnce_ConcurrentPublish_GlobalWildcard_FiresExactlyOnce(t *testing.T) {
	for attempt := 0; attempt < 200; attempt++ {
		b := New()
		var count atomic.Int64

		_, _ = b.Once("*", func(*Message) { count.Add(1) })

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = b.Publish("anything", nil)
			}()
		}
		wg.Wait()
		_ = b.Close()

		if n := count.Load(); n != 1 {
			t.Errorf("attempt %d: once handler fired %d times, want exactly 1", attempt, n)
			return
		}
	}
}

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

func TestSubscribeQueue_OnlyOneHandlerReceives(t *testing.T) {
	b := newBroker(t)
	var count atomic.Int64

	for i := 0; i < 5; i++ {
		_, _ = b.SubscribeQueue("task", "workers", func(*Message) { count.Add(1) })
	}

	_ = b.Publish("task", nil)

	if n := count.Load(); n != 1 {
		t.Error(test.DiffMessage(n, int64(1), "queue: exactly 1 handler should receive per publish"))
	}
}

func TestSubscribeQueue_DistributesAcrossWorkers(t *testing.T) {
	b := newBroker(t)
	counts := make([]atomic.Int64, 3)

	for i := range counts {
		i := i
		_, _ = b.SubscribeQueue("task", "workers", func(*Message) { counts[i].Add(1) })
	}

	const msgs = 30
	for i := 0; i < msgs; i++ {
		_ = b.Publish("task", nil)
	}

	total := int64(0)
	for i := range counts {
		n := counts[i].Load()
		if n == 0 {
			t.Errorf("worker %d received 0 messages — distribution is not working", i)
		}
		total += n
	}
	if total != msgs {
		t.Error(test.DiffMessage(total, int64(msgs), "total deliveries must equal publish count"))
	}
}

func TestSubscribeQueue_MultipleGroups_EachGetsOne(t *testing.T) {
	b := newBroker(t)
	var groupA, groupB atomic.Int64

	for i := 0; i < 3; i++ {
		_, _ = b.SubscribeQueue("task", "groupA", func(*Message) { groupA.Add(1) })
		_, _ = b.SubscribeQueue("task", "groupB", func(*Message) { groupB.Add(1) })
	}

	_ = b.Publish("task", nil)

	if groupA.Load() != 1 {
		t.Error(test.DiffMessage(groupA.Load(), int64(1), "groupA should receive exactly 1"))
	}
	if groupB.Load() != 1 {
		t.Error(test.DiffMessage(groupB.Load(), int64(1), "groupB should receive exactly 1"))
	}
}

func TestSubscribeQueue_Unsubscribe(t *testing.T) {
	b := newBroker(t)
	var count atomic.Int64

	sub, _ := b.SubscribeQueue("task", "workers", func(*Message) { count.Add(1) })
	_, _ = b.SubscribeQueue("task", "workers", func(*Message) { count.Add(1) })

	_ = sub.Unsubscribe()

	for i := 0; i < 20; i++ {
		_ = b.Publish("task", nil)
	}

	if n := count.Load(); n != 20 {
		t.Error(test.DiffMessage(n, int64(20), "after unsubscribe only 1 worker remains, should still handle all 20"))
	}
}

func TestSubscribeQueue_ListenerCount(t *testing.T) {
	b := newBroker(t)
	for i := 0; i < 4; i++ {
		_, _ = b.SubscribeQueue("task", "workers", func(*Message) {})
	}
	if n := b.ListenerCount("task"); n != 4 {
		t.Error(test.DiffMessage(n, 4, "ListenerCount should include queue subscribers"))
	}
}

func TestSubscribeQueue_EmptyGroup_ReturnsError(t *testing.T) {
	b := newBroker(t)
	_, err := b.SubscribeQueue("task", "", func(*Message) {})
	if err != ErrEmptyGroup {
		t.Error(test.DiffMessage(err, ErrEmptyGroup, "empty group should return ErrEmptyGroup"))
	}
}

func TestSubscribeQueue_Topics_Included(t *testing.T) {
	b := newBroker(t)
	_, _ = b.SubscribeQueue("task.process", "workers", func(*Message) {})

	topics := b.Topics()
	found := false
	for _, t := range topics {
		if t == "task.process" {
			found = true
		}
	}
	if !found {
		t.Error(test.DiffMessage(topics, []string{"task.process"}, "queue topic should appear in Topics()"))
	}
}

func TestSubscribeQueue_ConcurrentPublish(t *testing.T) {
	b := newBroker(t)
	var total atomic.Int64

	for i := 0; i < 5; i++ {
		_, _ = b.SubscribeQueue("task", "workers", func(*Message) { total.Add(1) })
	}

	var wg sync.WaitGroup
	const msgs = 100
	for i := 0; i < msgs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Publish("task", nil)
		}()
	}
	wg.Wait()

	if n := total.Load(); n != msgs {
		t.Error(test.DiffMessage(n, int64(msgs), "concurrent queue publish: total must equal message count"))
	}
}

func TestSubscribeQueue_FanOutAndQueueCoexist(t *testing.T) {
	b := newBroker(t)
	var fanOut, queue atomic.Int64

	_, _ = b.Subscribe("task", func(*Message) { fanOut.Add(1) })
	_, _ = b.Subscribe("task", func(*Message) { fanOut.Add(1) })
	for i := 0; i < 3; i++ {
		_, _ = b.SubscribeQueue("task", "workers", func(*Message) { queue.Add(1) })
	}

	_ = b.Publish("task", nil)

	if fanOut.Load() != 2 {
		t.Error(test.DiffMessage(fanOut.Load(), int64(2), "fan-out subs should all receive"))
	}
	if queue.Load() != 1 {
		t.Error(test.DiffMessage(queue.Load(), int64(1), "queue group should deliver to exactly 1"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Fix 1: round-robin starts at index 0
// ────────────────────────────────────────────────────────────────────────────

func TestSubscribeQueue_DistributesFromWorker0(t *testing.T) {
	// With the counter seeded to ^uint64(0), the first Add(1) wraps to 0,
	// so the first message always goes to worker 0.
	b := newBroker(t)
	counts := make([]atomic.Int64, 3)

	for i := range counts {
		i := i
		_, _ = b.SubscribeQueue("task", "workers", func(*Message) { counts[i].Add(1) })
	}

	const msgs = 30
	for range msgs {
		_ = b.Publish("task", nil)
	}

	// Every worker must receive at least one message (verified in the
	// existing TestSubscribeQueue_DistributesAcrossWorkers test). Additionally,
	// with 30 messages and 3 workers the distribution must be perfectly even.
	for i := range counts {
		n := counts[i].Load()
		if n == 0 {
			t.Errorf("worker %d received 0 messages — worker 0 is being skipped", i)
		}
	}
	// With 30 msgs and 3 workers the counter wraps evenly: each gets exactly 10.
	for i := range counts {
		if counts[i].Load() != 10 {
			t.Errorf("worker %d: got %d, want 10 (perfect round-robin)", i, counts[i].Load())
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Fix 2: empty bucket cleanup
// ────────────────────────────────────────────────────────────────────────────

func TestEmptyBucketCleanup_Exact(t *testing.T) {
	// Use the internal broker type to inspect the maps directly.
	b := NewWithConfig(Config{RecoverPanics: true}).(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	_, _ = b.Once("ephemeral.topic", func(*Message) {})
	_ = b.Publish("ephemeral.topic", nil) // fires once-sub → removed

	if len(b.Topics()) != 0 {
		t.Error(test.DiffMessage(b.Topics(), []string{}, "Topics() should be empty after once-sub fires"))
	}
	if n := b.ListenerCount("ephemeral.topic"); n != 0 {
		t.Error(test.DiffMessage(n, 0, "ListenerCount should be 0 after once-sub fires"))
	}

	// Verify the map entry itself is gone (no empty bucket).
	b.mu.RLock()
	_, hasBucket := b.exactByTopic["ephemeral.topic"]
	b.mu.RUnlock()
	if hasBucket {
		t.Error("empty exact bucket should have been deleted after once-sub cleanup")
	}
}

func TestEmptyBucketCleanup_Prefix(t *testing.T) {
	b := NewWithConfig(Config{RecoverPanics: true}).(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	_, _ = b.Once("order.*", func(*Message) {})
	_ = b.Publish("order.created", nil)

	b.mu.RLock()
	_, hasBucket := b.prefixByPrefix["order"]
	b.mu.RUnlock()
	if hasBucket {
		t.Error("empty prefix bucket should have been deleted after once-sub cleanup")
	}
}

func TestEmptyBucketCleanup_Unsubscribe(t *testing.T) {
	b := NewWithConfig(Config{RecoverPanics: true}).(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	sub, _ := b.Subscribe("only.sub", func(*Message) {})
	_ = b.Unsubscribe(sub)

	b.mu.RLock()
	_, hasBucket := b.exactByTopic["only.sub"]
	b.mu.RUnlock()
	if hasBucket {
		t.Error("empty exact bucket should have been deleted after unsubscribe")
	}
}

func TestEmptyBucketCleanup_QueueGroup(t *testing.T) {
	b := NewWithConfig(Config{RecoverPanics: true}).(*MemoryBroker)
	t.Cleanup(func() { _ = b.Close() })

	sub, _ := b.SubscribeQueue("q.topic", "grp", func(*Message) {})
	_ = b.Unsubscribe(sub)

	b.mu.RLock()
	_, hasTopic := b.queueGroupsByTopic["q.topic"]
	b.mu.RUnlock()
	if hasTopic {
		t.Error("empty queueGroups topic entry should have been deleted after unsubscribe")
	}
	if len(b.Topics()) != 0 {
		t.Errorf("Topics() should be empty, got %v", b.Topics())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Fix 3: panic recovery
// ────────────────────────────────────────────────────────────────────────────

func TestPanicRecovery_OtherHandlersStillReceive(t *testing.T) {
	b := NewWithConfig(Config{RecoverPanics: true})
	t.Cleanup(func() { _ = b.Close() })

	var after atomic.Int64
	_, _ = b.Subscribe("panic.topic", func(*Message) { panic("deliberate") })
	_, _ = b.Subscribe("panic.topic", func(*Message) { after.Add(1) })
	_, _ = b.Subscribe("panic.topic", func(*Message) { after.Add(1) })

	_ = b.Publish("panic.topic", nil)

	if n := after.Load(); n != 2 {
		t.Errorf("handlers after the panicking one: got %d, want 2", n)
	}
}

func TestPanicRecovery_OnPanicCalled(t *testing.T) {
	var panicMsg *Message
	var panicVal any
	var mu sync.Mutex

	b := NewWithConfig(Config{
		RecoverPanics: true,
		OnPanic: func(m *Message, r any) {
			mu.Lock()
			panicMsg = m
			panicVal = r
			mu.Unlock()
		},
	})
	t.Cleanup(func() { _ = b.Close() })

	_, _ = b.Subscribe("oops", func(*Message) { panic("bad handler") })
	_ = b.Publish("oops", "data")

	mu.Lock()
	defer mu.Unlock()
	if panicMsg == nil {
		t.Fatal("OnPanic was not called")
	}
	if panicMsg.Topic != "oops" {
		t.Errorf("OnPanic: topic = %q, want %q", panicMsg.Topic, "oops")
	}
	if panicVal != "bad handler" {
		t.Errorf("OnPanic: recovered value = %v, want %q", panicVal, "bad handler")
	}
}

func TestPanicRecovery_Disabled(t *testing.T) {
	b := NewWithConfig(Config{RecoverPanics: false})
	t.Cleanup(func() { _ = b.Close() })

	_, _ = b.Subscribe("boom", func(*Message) { panic("unrecovered") })

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic to propagate when RecoverPanics: false")
		}
	}()
	_ = b.Publish("boom", nil)
}

// ────────────────────────────────────────────────────────────────────────────
// Fix 4: bounded async worker pool
// ────────────────────────────────────────────────────────────────────────────

func TestPublishAsync_WorkerPool_DeliversAll(t *testing.T) {
	b := NewWithConfig(Config{
		RecoverPanics:  true,
		AsyncWorkers:   4,
		AsyncQueueSize: 200,
	})
	t.Cleanup(func() { _ = b.Close() })

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
	t.Errorf("worker pool: got %d deliveries, want %d", count.Load(), msgs)
}

func TestPublishAsync_QueueFull_ReturnsError(t *testing.T) {
	// Use 1 worker and a tiny queue. Fill the queue before the worker can drain.
	// We block the worker with a channel so the queue stays full.
	block := make(chan struct{})
	b := NewWithConfig(Config{
		RecoverPanics:  true,
		AsyncWorkers:   1,
		AsyncQueueSize: 2,
	})
	t.Cleanup(func() { _ = b.Close() })

	_, _ = b.Subscribe("full.topic", func(*Message) {
		<-block // block all workers
	})

	// Send one message to block the single worker.
	_ = b.PublishAsync("full.topic", "block")
	// Give the worker time to pick up the job and start blocking.
	time.Sleep(20 * time.Millisecond)

	// Now fill the remaining queue capacity.
	_ = b.PublishAsync("full.topic", "fill1")
	_ = b.PublishAsync("full.topic", "fill2")

	// Next send must fail.
	err := b.PublishAsync("full.topic", "overflow")
	if err != ErrAsyncQueueFull {
		t.Errorf("expected ErrAsyncQueueFull, got %v", err)
	}

	close(block) // unblock workers so Close() can drain
}

func TestClose_WorkerPool_Drains(t *testing.T) {
	var count atomic.Int64
	b := NewWithConfig(Config{
		RecoverPanics:  true,
		AsyncWorkers:   2,
		AsyncQueueSize: 50,
	})

	_, _ = b.Subscribe("drain.topic", func(*Message) {
		time.Sleep(1 * time.Millisecond)
		count.Add(1)
	})

	const msgs = 20
	for i := range msgs {
		_ = b.PublishAsync("drain.topic", i)
	}

	// Close must block until all enqueued jobs finish.
	_ = b.Close()

	if count.Load() != msgs {
		t.Errorf("after Close: delivered %d, want %d — worker pool did not drain", count.Load(), msgs)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Fix 5: broker statistics
// ────────────────────────────────────────────────────────────────────────────

func TestStats_CountsCorrectly(t *testing.T) {
	b := NewWithConfig(Config{RecoverPanics: true})
	t.Cleanup(func() { _ = b.Close() })

	noop := func(*Message) {}
	_, _ = b.Subscribe("s.one", noop)
	_, _ = b.Subscribe("s.one", noop)
	_, _ = b.Subscribe("s.*", noop)
	_, _ = b.SubscribeQueue("s.one", "grp", noop)

	_ = b.Publish("s.one", nil)
	_ = b.Publish("s.one", nil)

	st := b.Stats()

	// Topics: "s.one" (exact+queue share one topic key), "s.*"
	if st.Topics != 2 {
		t.Errorf("Topics: got %d, want 2", st.Topics)
	}
	// Subscribers: 2 exact + 1 queue + 1 prefix = 4
	if st.Subscribers != 4 {
		t.Errorf("Subscribers: got %d, want 4", st.Subscribers)
	}
	if st.PublishCalls != 2 {
		t.Errorf("PublishCalls: got %d, want 2", st.PublishCalls)
	}
	// Each Publish to "s.one" delivers to: 2 exact + 1 prefix + 1 queue = 4
	if st.MessagesSent != 8 {
		t.Errorf("MessagesSent: got %d, want 8", st.MessagesSent)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Fix 6: observability hooks
// ────────────────────────────────────────────────────────────────────────────

func TestHooks_AllFourFire(t *testing.T) {
	var (
		beforePublishTopic string
		afterPublishTopic  string
		beforeDispatch     []int
		afterDispatch      []int
		mu                 sync.Mutex
	)

	b := NewWithConfig(Config{
		RecoverPanics: true,
		BeforePublish: func(topic string, _ any) {
			mu.Lock()
			beforePublishTopic = topic
			mu.Unlock()
		},
		AfterPublish: func(topic string, _ any, _ error) {
			mu.Lock()
			afterPublishTopic = topic
			mu.Unlock()
		},
		BeforeDispatch: func(_ *Message, idx int) {
			mu.Lock()
			beforeDispatch = append(beforeDispatch, idx)
			mu.Unlock()
		},
		AfterDispatch: func(_ *Message, idx int) {
			mu.Lock()
			afterDispatch = append(afterDispatch, idx)
			mu.Unlock()
		},
	})
	t.Cleanup(func() { _ = b.Close() })

	noop := func(*Message) {}
	_, _ = b.Subscribe("hook.topic", noop)
	_, _ = b.Subscribe("hook.topic", noop)

	_ = b.Publish("hook.topic", "x")

	mu.Lock()
	defer mu.Unlock()

	if beforePublishTopic != "hook.topic" {
		t.Errorf("BeforePublish: topic = %q, want %q", beforePublishTopic, "hook.topic")
	}
	if afterPublishTopic != "hook.topic" {
		t.Errorf("AfterPublish: topic = %q, want %q", afterPublishTopic, "hook.topic")
	}
	if len(beforeDispatch) != 2 {
		t.Errorf("BeforeDispatch called %d times, want 2", len(beforeDispatch))
	}
	if len(afterDispatch) != 2 {
		t.Errorf("AfterDispatch called %d times, want 2", len(afterDispatch))
	}
	// Indices should be 0 and 1 in order.
	for i, idx := range beforeDispatch {
		if idx != i {
			t.Errorf("BeforeDispatch[%d] = %d, want %d", i, idx, i)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// NewWithConfig: Broker interface satisfied
// ────────────────────────────────────────────────────────────────────────────

func TestNewWithConfig_ImplementsBroker(t *testing.T) {
	// Compile-time interface check: both constructors must satisfy Broker.
	_ = []Broker{NewWithConfig(Config{}), New()}
}

func TestPublishAsync_Close_NoPanic(t *testing.T) {
	for i := 0; i < 2000; i++ {
		b := NewWithConfig(Config{
			RecoverPanics:  true,
			AsyncWorkers:   4,
			AsyncQueueSize: 8,
		})
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

func TestHook_BeforePublish_Panic_DeliveryNotAborted(t *testing.T) {
	var received atomic.Int64
	b := NewWithConfig(Config{
		RecoverPanics: true,
		BeforePublish: func(string, any) { panic("before-publish boom") },
	})
	defer func() { _ = b.Close() }()
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })

	_ = b.Publish("t", nil)

	if received.Load() != 1 {
		t.Error(test.DiffMessage(received.Load(), int64(1), "BeforePublish panic must not abort delivery"))
	}
}

func TestHook_AfterPublish_Panic_DoesNotCrash(t *testing.T) {
	var received atomic.Int64
	b := NewWithConfig(Config{
		RecoverPanics: true,
		AfterPublish:  func(string, any, error) { panic("after-publish boom") },
	})
	defer func() { _ = b.Close() }()
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })

	_ = b.Publish("t", nil)

	if received.Load() != 1 {
		t.Error(test.DiffMessage(received.Load(), int64(1), "AfterPublish panic must not abort delivery"))
	}
}

func TestHook_BeforeDispatch_Panic_OtherSubscribersStillReceive(t *testing.T) {
	var received atomic.Int64
	b := NewWithConfig(Config{
		RecoverPanics:  true,
		BeforeDispatch: func(*Message, int) { panic("before-dispatch boom") },
	})
	defer func() { _ = b.Close() }()
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })

	_ = b.Publish("t", nil)

	if received.Load() != 3 {
		t.Error(test.DiffMessage(received.Load(), int64(3), "BeforeDispatch panic must not skip subscribers"))
	}
}

func TestHook_AfterDispatch_Panic_OtherSubscribersStillReceive(t *testing.T) {
	var received atomic.Int64
	b := NewWithConfig(Config{
		RecoverPanics: true,
		AfterDispatch: func(*Message, int) { panic("after-dispatch boom") },
	})
	defer func() { _ = b.Close() }()
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })

	_ = b.Publish("t", nil)

	if received.Load() != 3 {
		t.Error(test.DiffMessage(received.Load(), int64(3), "AfterDispatch panic must not skip subscribers"))
	}
}

func TestHook_OnPanic_Panic_DoesNotCrash(t *testing.T) {
	var received atomic.Int64
	b := NewWithConfig(Config{
		RecoverPanics: true,
		OnPanic:       func(*Message, any) { panic("on-panic boom") },
	})
	defer func() { _ = b.Close() }()
	_, _ = b.Subscribe("t", func(*Message) { panic("subscriber boom") })
	_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })

	_ = b.Publish("t", nil)

	if received.Load() != 1 {
		t.Error(test.DiffMessage(received.Load(), int64(1), "OnPanic panic must not crash broker or skip other subscribers"))
	}
}

func TestHook_AllPanic_AllSubscribersStillReceive(t *testing.T) {
	var received atomic.Int64
	b := NewWithConfig(Config{
		RecoverPanics:  true,
		BeforePublish:  func(string, any) { panic("bp") },
		AfterPublish:   func(string, any, error) { panic("ap") },
		BeforeDispatch: func(*Message, int) { panic("bd") },
		AfterDispatch:  func(*Message, int) { panic("ad") },
		OnPanic:        func(*Message, any) { panic("op") },
	})
	defer func() { _ = b.Close() }()
	for i := 0; i < 5; i++ {
		_, _ = b.Subscribe("t", func(*Message) { received.Add(1) })
	}

	_ = b.Publish("t", nil)

	if received.Load() != 5 {
		t.Error(test.DiffMessage(received.Load(), int64(5), "all hooks panicking must not drop any subscriber delivery"))
	}
}

func TestSubscribe_MultiLevel_MatchesDeepTopics(t *testing.T) {
	b := newBroker(t)
	var received []string
	_, _ = b.Subscribe("user.>", func(m *Message) { received = append(received, m.Topic) })

	_ = b.Publish("user.created", nil)
	_ = b.Publish("user.profile.updated", nil)
	_ = b.Publish("user.profile.avatar.changed", nil)
	_ = b.Publish("order.created", nil)

	if len(received) != 3 {
		t.Error(test.DiffMessage(len(received), 3, "user.> should match 3 deep topics"))
	}
}

func TestSubscribe_MultiLevel_DoesNotMatchParent(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("user.>", func(*Message) { count++ })

	_ = b.Publish("user", nil)

	if count != 0 {
		t.Error(test.DiffMessage(count, 0, "user.> must not match 'user' itself"))
	}
}

func TestSubscribe_GlobalMulti_MatchesAll(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe(">", func(*Message) { count++ })

	_ = b.Publish("a", nil)
	_ = b.Publish("a.b", nil)
	_ = b.Publish("a.b.c", nil)

	if count != 3 {
		t.Error(test.DiffMessage(count, 3, "> should match every published topic"))
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

func TestSubscribe_Complex_Mixed(t *testing.T) {
	b := newBroker(t)
	var received []string
	_, _ = b.Subscribe("tenant.*.user.>", func(m *Message) { received = append(received, m.Topic) })

	_ = b.Publish("tenant.1.user.created", nil)
	_ = b.Publish("tenant.abc.user.profile.updated", nil)
	_ = b.Publish("tenant.1.admin.created", nil)
	_ = b.Publish("tenant.1.user", nil)

	if len(received) != 2 {
		t.Error(test.DiffMessage(len(received), 2, "tenant.*.user.> should match 2 topics"))
	}
}

func TestOff_Complex_Pattern(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("a.*.b.>", func(*Message) { count++ })

	_ = b.Publish("a.x.b.y", nil)
	_ = b.Off("a.*.b.>")
	_ = b.Publish("a.x.b.y", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "Off(complex pattern) should remove all its subscribers"))
	}
}

func TestSubscribeQueue_Wildcard_ReturnsError(t *testing.T) {
	b := newBroker(t)
	_, err := b.SubscribeQueue("user.*", "workers", func(*Message) {})
	if err != ErrWildcardInQueue {
		t.Error(test.DiffMessage(err, ErrWildcardInQueue, "wildcard topic in SubscribeQueue should return ErrWildcardInQueue"))
	}
}

func TestSubscribe_BackwardCompat_StarAlias(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("*", func(*Message) { count++ })

	_ = b.Publish("a.b.c", nil)
	_ = b.Publish("x", nil)

	if count != 2 {
		t.Error(test.DiffMessage(count, 2, "* should be an alias for > and match all topics"))
	}
}

func TestSubscribe_BackwardCompat_UserStar(t *testing.T) {
	b := newBroker(t)
	var count int
	_, _ = b.Subscribe("user.*", func(*Message) { count++ })

	_ = b.Publish("user.created", nil)
	_ = b.Publish("user.profile.updated", nil)

	if count != 1 {
		t.Error(test.DiffMessage(count, 1, "user.* should still match only one level"))
	}
}
