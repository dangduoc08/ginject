package broker2

import (
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
)

// ────────────────────────────────────────────────────────────────────────────
// Subscribe
// ────────────────────────────────────────────────────────────────────────────

func TestSubscribe_EmptyTopic_ReturnsError(t *testing.T) {
	b := NewBroker()

	id, err := b.Subscribe("", func(*Message) {})
	if err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "Subscribe with empty topic"))
	}
	if id != 0 {
		t.Error(test.DiffMessage(id, 0, "Subscribe with empty topic should return a zero id"))
	}
}

func TestSubscribe_NilHandler_ReturnsError(t *testing.T) {
	b := NewBroker()

	id, err := b.Subscribe("foo", nil)
	if err != ErrNilHandler {
		t.Error(test.DiffMessage(err, ErrNilHandler, "Subscribe with nil handler"))
	}
	if id != 0 {
		t.Error(test.DiffMessage(id, 0, "Subscribe with nil handler should return a zero id"))
	}
}

func TestSubscribe_Success_ReturnsNonZeroID(t *testing.T) {
	b := NewBroker()

	id, err := b.Subscribe("foo", func(*Message) {})
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "Subscribe should not error on valid input"))
	}
	if id == 0 {
		t.Error(test.DiffMessage(id, "non-zero", "Subscribe should return a non-zero id"))
	}
}

func TestSubscribe_TopicWithSpecialCharacters_NoPanic(t *testing.T) {
	b := NewBroker()

	topics := []string{
		"../../etc/passwd",
		"foo\x00bar",
		"'; DROP TABLE subs; --",
		strings.Repeat("a.", 1000) + "z",
	}
	for _, topic := range topics {
		if _, err := b.Subscribe(topic, func(*Message) {}); err != nil {
			t.Error(test.DiffMessage(err, nil, "Subscribe must accept topic: "+topic))
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Publish
// ────────────────────────────────────────────────────────────────────────────

func TestPublish_EmptyTopic_ReturnsError(t *testing.T) {
	b := NewBroker()

	if err := b.Publish("", "payload"); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "Publish with empty topic"))
	}
}

func TestPublish_NoSubscribers_NoOp(t *testing.T) {
	b := NewBroker()

	if err := b.Publish("nobody.listening", "payload"); err != nil {
		t.Error(test.DiffMessage(err, nil, "Publish with zero subscribers should not error"))
	}
}

func TestPublish_DeliversToSubscribedHandler(t *testing.T) {
	b := NewBroker()

	var got *Message
	_, _ = b.Subscribe("foo.bar", func(m *Message) { got = m })

	if err := b.Publish("foo.bar", "x"); err != nil {
		t.Error(test.DiffMessage(err, nil, "Publish should not error"))
	}
	if got == nil || got.Payload != "x" || got.Topic != "foo.bar" {
		t.Error(test.DiffMessage(got, "&Message{Topic: \"foo.bar\", Payload: \"x\"}", "Publish should deliver topic and payload to the handler"))
	}
}

func TestPublish_HandlerPanic_DoesNotCrashOrBlockSiblings(t *testing.T) {
	b := NewBroker()

	_, _ = b.Subscribe("foo.bar", func(*Message) { panic("boom") })
	var calledAfterPanic bool
	_, _ = b.Subscribe("foo.bar", func(*Message) { calledAfterPanic = true })

	if err := b.Publish("foo.bar", nil); err != nil {
		t.Error(test.DiffMessage(err, nil, "Publish should not error even if a handler panics"))
	}
	if !calledAfterPanic {
		t.Error(test.DiffMessage(calledAfterPanic, true, "a panicking handler must not stop sibling handlers from running"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// PublishAsync
// ────────────────────────────────────────────────────────────────────────────

func TestPublishAsync_EmptyTopic_ReturnsError(t *testing.T) {
	b := NewBroker()

	if err := b.PublishAsync("", "payload"); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "PublishAsync with empty topic"))
	}
}

func TestPublishAsync_NoSubscribers_NoOp(t *testing.T) {
	b := NewBroker()

	if err := b.PublishAsync("nobody.listening", "payload"); err != nil {
		t.Error(test.DiffMessage(err, nil, "PublishAsync with zero subscribers should not error"))
	}
}

func TestPublishAsync_DeliversToSubscribedHandler(t *testing.T) {
	b := NewBroker()

	done := make(chan *Message, 1)
	_, _ = b.Subscribe("foo.bar", func(m *Message) { done <- m })

	if err := b.PublishAsync("foo.bar", "x"); err != nil {
		t.Error(test.DiffMessage(err, nil, "PublishAsync should not error"))
	}

	select {
	case got := <-done:
		if got.Payload != "x" || got.Topic != "foo.bar" {
			t.Error(test.DiffMessage(got, "&Message{Topic: \"foo.bar\", Payload: \"x\"}", "PublishAsync should deliver topic and payload to the handler"))
		}
	case <-time.After(time.Second):
		t.Fatal("handler was not invoked within 1s")
	}
}

func TestPublishAsync_HandlerPanic_DoesNotCrashOrBlockSiblings(t *testing.T) {
	b := NewBroker()

	done := make(chan struct{}, 1)
	_, _ = b.Subscribe("foo.bar", func(*Message) { panic("boom") })
	_, _ = b.Subscribe("foo.bar", func(*Message) { done <- struct{}{} })

	if err := b.PublishAsync("foo.bar", nil); err != nil {
		t.Error(test.DiffMessage(err, nil, "PublishAsync should not error even if a handler panics"))
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("sibling handler did not run within 1s after a panicking handler")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Unsubscribe
// ────────────────────────────────────────────────────────────────────────────

func TestUnsubscribe_EmptyTopic_ReturnsError(t *testing.T) {
	b := NewBroker()

	if err := b.Unsubscribe("", 1); err != ErrEmptyTopic {
		t.Error(test.DiffMessage(err, ErrEmptyTopic, "Unsubscribe with empty topic"))
	}
}

func TestUnsubscribe_UnknownTopicOrID_NoError(t *testing.T) {
	b := NewBroker()

	if err := b.Unsubscribe("never.subscribed", 999); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe of an unknown topic/id should not error"))
	}
}

func TestUnsubscribe_RemovesOnlyThatHandler(t *testing.T) {
	b := NewBroker()

	var calledA, calledB bool
	idA, _ := b.Subscribe("foo.bar", func(*Message) { calledA = true })
	_, _ = b.Subscribe("foo.bar", func(*Message) { calledB = true })

	if err := b.Unsubscribe("foo.bar", idA); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe should not error"))
	}

	_ = b.Publish("foo.bar", nil)

	if calledA || !calledB {
		t.Error(test.DiffMessage(
			[]bool{calledA, calledB},
			[]bool{false, true},
			"Unsubscribe should remove only the targeted handler",
		))
	}
}

func TestUnsubscribe_DoesNotAffectOtherTopics(t *testing.T) {
	b := NewBroker()

	var calledOther bool
	idX, _ := b.Subscribe("topic.x", func(*Message) {})
	_, _ = b.Subscribe("topic.y", func(*Message) { calledOther = true })

	_ = b.Unsubscribe("topic.x", idX)
	_ = b.Publish("topic.y", nil)

	if !calledOther {
		t.Error(test.DiffMessage(calledOther, true, "Unsubscribe on one topic must not affect handlers on another topic"))
	}
}

func TestUnsubscribe_DoubleUnsubscribe_NoError(t *testing.T) {
	b := NewBroker()

	id, _ := b.Subscribe("foo", func(*Message) {})

	if err := b.Unsubscribe("foo", id); err != nil {
		t.Error(test.DiffMessage(err, nil, "first Unsubscribe should not error"))
	}
	if err := b.Unsubscribe("foo", id); err != nil {
		t.Error(test.DiffMessage(err, nil, "second Unsubscribe of the same id should not error"))
	}
}

func TestUnsubscribe_AfterUnsubscribe_PublishSkipsHandler(t *testing.T) {
	b := NewBroker()

	calls := 0
	id, _ := b.Subscribe("foo", func(*Message) { calls++ })

	_ = b.Publish("foo", nil)
	_ = b.Unsubscribe("foo", id)
	_ = b.Publish("foo", nil)

	if calls != 1 {
		t.Error(test.DiffMessage(calls, 1, "handler should not fire for any Publish after Unsubscribe"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Subscriptions
// ────────────────────────────────────────────────────────────────────────────

func TestSubscriptions_EmptyBroker_ReturnsEmptyMap(t *testing.T) {
	b := NewBroker()

	if subs := b.Subscriptions(); len(subs) != 0 {
		t.Error(test.DiffMessage(subs, map[string][]uint64{}, "Subscriptions on a fresh broker should be empty"))
	}
}

func TestSubscriptions_ListsTopicsAndIDs(t *testing.T) {
	b := NewBroker()

	idA, _ := b.Subscribe("foo.bar", func(*Message) {})
	idB, _ := b.Subscribe("foo.bar", func(*Message) {})
	idC, _ := b.Subscribe("baz", func(*Message) {})

	subs := b.Subscriptions()
	if len(subs) != 2 {
		t.Fatal(test.DiffMessage(len(subs), 2, "Subscriptions should report one entry per topic"))
	}

	fooIDs := subs["foo.bar"]
	if len(fooIDs) != 2 || !slices.Contains(fooIDs, idA) || !slices.Contains(fooIDs, idB) {
		t.Error(test.DiffMessage(fooIDs, []uint64{idA, idB}, "Subscriptions[\"foo.bar\"] should contain both ids"))
	}

	bazIDs := subs["baz"]
	if len(bazIDs) != 1 || bazIDs[0] != idC {
		t.Error(test.DiffMessage(bazIDs, []uint64{idC}, "Subscriptions[\"baz\"] should contain its id"))
	}
}

func TestSubscriptions_AfterUnsubscribe_RemovesEntry(t *testing.T) {
	b := NewBroker()

	id, _ := b.Subscribe("foo", func(*Message) {})
	_ = b.Unsubscribe("foo", id)

	if subs := b.Subscriptions(); len(subs) != 0 {
		t.Error(test.DiffMessage(subs, map[string][]uint64{}, "Subscriptions must not list a topic after its last handler is removed"))
	}
}

func TestSubscriptions_ReturnsIndependentCopy(t *testing.T) {
	b := NewBroker()
	_, _ = b.Subscribe("foo", func(*Message) {})

	subs := b.Subscriptions()
	_, _ = b.Subscribe("foo", func(*Message) {})

	if len(subs["foo"]) != 1 {
		t.Error(test.DiffMessage(len(subs["foo"]), 1, "snapshot returned by Subscriptions must not observe later inserts"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Concurrent safety (run with: go test -race ./broker/...)
// ────────────────────────────────────────────────────────────────────────────

func TestConcurrentSubscribeUnsubscribePublish(t *testing.T) {
	b := NewBroker()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			id, _ := b.Subscribe("race.topic", func(*Message) {})
			_ = b.Unsubscribe("race.topic", id)
		}()
		go func() {
			defer wg.Done()
			_ = b.Publish("race.topic", nil)
		}()
		go func() {
			defer wg.Done()
			_ = b.Subscriptions()
		}()
	}
	wg.Wait()
}

func TestConcurrentPublishAsync(t *testing.T) {
	b := NewBroker()

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	_, _ = b.Subscribe("race.topic", func(*Message) { wg.Done() })

	for i := 0; i < n; i++ {
		_ = b.PublishAsync("race.topic", nil)
	}
	wg.Wait()
}
