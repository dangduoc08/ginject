package memorybroker

import (
	"fmt"
	"testing"
)

// BenchmarkPublish measures synchronous publish to a single exact-match topic
// with 1000 subscribers pre-registered.
func BenchmarkPublish(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

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

// BenchmarkPublishWildcard measures publish when 100 wildcard subscribers are registered.
func BenchmarkPublishWildcard(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

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

// BenchmarkPublishMixed measures publish when exact, prefix, and global
// subscribers are all active simultaneously.
func BenchmarkPublishMixed(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

	noop := func(_ *Message) {}
	for i := 0; i < 10; i++ {
		_, _ = br.Subscribe("mixed.topic", noop)
		_, _ = br.Subscribe("mixed.*", noop)
		_, _ = br.Subscribe("*", noop)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Publish("mixed.topic", i)
	}
}

// BenchmarkSubscribe measures the cost of subscribe followed by unsubscribe in a tight loop.
func BenchmarkSubscribe(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

	noop := func(_ *Message) {}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		sub, _ := br.Subscribe("bench.sub", noop)
		_ = br.Unsubscribe(sub)
	}
}

// BenchmarkPublishParallel measures throughput under concurrent publish load.
func BenchmarkPublishParallel(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

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

// BenchmarkPublishComplex measures publish against complex (middle-wildcard) patterns.
func BenchmarkPublishComplex(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

	noop := func(_ *Message) {}
	for i := 0; i < 10; i++ {
		_, _ = br.Subscribe(fmt.Sprintf("tenant.*.user.%d", i), noop)
	}
	_, _ = br.Subscribe("tenant.*.user.created", noop)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Publish("tenant.abc.user.created", i)
	}
}

// BenchmarkPublishNoSubscribers measures publish to a topic with zero matching subscribers.
func BenchmarkPublishNoSubscribers(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

	noop := func(_ *Message) {}
	_, _ = br.Subscribe("other.topic", noop)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Publish("nobody.listens", i)
	}
}

// BenchmarkSubscribe_OnClosedBroker measures the cost of Subscribe when the
// broker is already closed and every call is rejected.
func BenchmarkSubscribe_OnClosedBroker(b *testing.B) {
	br := NewMemoryBroker()
	_ = br.Close()

	noop := func(_ *Message) {}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = br.Subscribe("bench.sub", noop)
	}
}

// BenchmarkUnsubscribe_Nil measures the cost of the nil-Subscription no-op path.
func BenchmarkUnsubscribe_Nil(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Unsubscribe(nil)
	}
}

// BenchmarkUnsubscribe_NilParallel measures the nil-Subscription no-op path
// under concurrent contention, where lock-free fast paths matter most.
func BenchmarkUnsubscribe_NilParallel(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = br.Unsubscribe(nil)
		}
	})
}

// BenchmarkPublishManyTopics measures publish when subscriptions are spread across many topics.
func BenchmarkPublishManyTopics(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

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
