package memorybroker

import (
	"fmt"
	"testing"
)

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

func BenchmarkUnsubscribe_Nil(b *testing.B) {
	br := NewMemoryBroker()
	defer func() { _ = br.Close() }()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = br.Unsubscribe(nil)
	}
}

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
