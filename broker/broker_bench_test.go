package broker

import (
	"fmt"
	"testing"
)

// BenchmarkPublish measures synchronous publish to a single exact-match topic
// with 1000 subscribers pre-registered.
func BenchmarkPublish(b *testing.B) {
	br := New()
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
	br := New()
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
	br := New()
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
	br := New()
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
	br := New()
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

// BenchmarkPublishManyTopics measures publish when subscriptions are spread across many topics.
func BenchmarkPublishManyTopics(b *testing.B) {
	br := New()
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
