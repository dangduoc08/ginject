package broker2

import (
	"fmt"
	"testing"
)

// BenchmarkPublish measures synchronous publish to a single exact-match topic
// with 1000 subscribers pre-registered.
func BenchmarkPublish(b *testing.B) {
	br := NewBroker()

	noop := func(_ *Message) {}
	for i := 0; i < 1000; i++ {
		_, _ = br.Subscribe("bench.topic", noop)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Publish("bench.topic", i)
	}
}

// BenchmarkPublishNoSubscribers measures publish to a topic with zero subscribers.
func BenchmarkPublishNoSubscribers(b *testing.B) {
	br := NewBroker()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Publish("nobody.listening", i)
	}
}

// BenchmarkPublishAsync measures the synchronous cost of PublishAsync (validation + goroutine spawn).
func BenchmarkPublishAsync(b *testing.B) {
	br := NewBroker()

	noop := func(_ *Message) {}
	for i := 0; i < 1000; i++ {
		_, _ = br.Subscribe("bench.topic", noop)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.PublishAsync("bench.topic", i)
	}
}

// BenchmarkPublishWildcard measures publish when 100 wildcard subscribers are registered.
func BenchmarkPublishWildcard(b *testing.B) {
	br := NewBroker()

	noop := func(_ *Message) {}
	for i := 0; i < 100; i++ {
		_, _ = br.Subscribe("*", noop)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Publish("any.topic", i)
	}
}

// BenchmarkPublishManyTopics measures publish when subscriptions are spread across many topics.
func BenchmarkPublishManyTopics(b *testing.B) {
	br := NewBroker()

	noop := func(_ *Message) {}
	for i := 0; i < 1000; i++ {
		_, _ = br.Subscribe(fmt.Sprintf("topic.%d", i), noop)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Publish(fmt.Sprintf("topic.%d", i%1000), i)
	}
}

// BenchmarkSubscriptions measures the cost of snapshotting topic->ids across many topics.
func BenchmarkSubscriptions(b *testing.B) {
	br := NewBroker()

	noop := func(_ *Message) {}
	for i := 0; i < 1000; i++ {
		_, _ = br.Subscribe(fmt.Sprintf("topic.%d", i), noop)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Subscriptions()
	}
}

// BenchmarkSubscribeUnsubscribe measures the cost of subscribe followed by unsubscribe in a tight loop.
func BenchmarkSubscribeUnsubscribe(b *testing.B) {
	br := NewBroker()

	noop := func(_ *Message) {}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		id, _ := br.Subscribe("bench.sub", noop)
		_ = br.Unsubscribe("bench.sub", id)
	}
}

// BenchmarkPublishParallel measures throughput under concurrent publish load.
func BenchmarkPublishParallel(b *testing.B) {
	br := NewBroker()

	noop := func(_ *Message) {}
	for i := 0; i < 10; i++ {
		_, _ = br.Subscribe("parallel.topic", noop)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_ = br.Publish("parallel.topic", i)
			i++
		}
	})
}
