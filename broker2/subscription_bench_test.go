package broker2

import (
	"strconv"
	"testing"
)

const benchTopicCount = 1000

func seedSubscription() *Subscription {
	s := NewSubscription()
	for i := 0; i < benchTopicCount; i++ {
		s.insert("svc.events."+strconv.Itoa(i), func(*Message) {})
	}
	return s
}

func BenchmarkInsert_NewTopic(b *testing.B) {
	s := seedSubscription()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.insert("svc.events.new."+strconv.Itoa(i), func(*Message) {})
	}
}

func BenchmarkInsert_ExistingTopic(b *testing.B) {
	s := seedSubscription()
	handler := func(*Message) {}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.insert("svc.events.500", handler)
	}
}

func BenchmarkFind_ExactMatch(b *testing.B) {
	s := seedSubscription()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.find("svc.events.500")
	}
}

func BenchmarkFind_NoMatch(b *testing.B) {
	s := seedSubscription()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.find("svc.events.unregistered")
	}
}

func BenchmarkFind_WildcardMatch(b *testing.B) {
	s := seedSubscription()
	s.insert("svc.events.*", func(*Message) {})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.find("svc.events.unregistered")
	}
}

func BenchmarkInsertAndRemove(b *testing.B) {
	s := seedSubscription()
	handler := func(*Message) {}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := s.insert("bench.remove.topic", handler)
		s.remove("bench.remove.topic", id)
	}
}

func BenchmarkList_Empty(b *testing.B) {
	s := NewSubscription()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.list()
	}
}

func BenchmarkList_Populated(b *testing.B) {
	s := seedSubscription()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = s.list()
	}
}

func BenchmarkFind_ExactMatch_Parallel(b *testing.B) {
	s := seedSubscription()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = s.find("svc.events.500")
		}
	})
}
