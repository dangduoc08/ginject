package wsevent_test

import (
	"fmt"
	"testing"

	"github.com/dangduoc08/ginject/wsevent"
)

func newBenchWSEvent(exactPatterns, wildcardPatterns int) *wsevent.WSEvent {
	m := wsevent.NewWSEvent()
	for i := 0; i < exactPatterns; i++ {
		m.Add(fmt.Sprintf("chat.room%d.message", i), wsevent.WSEventItem{Handler: i})
	}
	for i := 0; i < wildcardPatterns; i++ {
		m.Add(fmt.Sprintf("chat.room%d.*", i), wsevent.WSEventItem{Handler: i})
	}
	return m
}

func BenchmarkWSEvent_Add(b *testing.B) {
	m := wsevent.NewWSEvent()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Add(fmt.Sprintf("chat.room%d.message", i%2000), wsevent.WSEventItem{Handler: i})
	}
}

func BenchmarkWSEvent_MatchExact(b *testing.B) {
	m := newBenchWSEvent(1000, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match("chat.room500.message")
	}
}

func BenchmarkWSEvent_MatchWildcard(b *testing.B) {
	m := newBenchWSEvent(1000, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match("chat.room1500.anything")
	}
}

func BenchmarkWSEvent_MatchNoMatch(b *testing.B) {
	m := newBenchWSEvent(1000, 1000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match("unregistered.topic.here")
	}
}
