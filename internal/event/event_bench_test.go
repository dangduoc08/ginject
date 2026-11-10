package event

import (
	"sync/atomic"
	"testing"
)

var parallelHits atomic.Int64

type traceLikeEvent struct {
	ID, Stage, Name, Transport string
	Duration                   int64
}

func BenchmarkEvent_HasListeners_TracingOff(b *testing.B) {
	e := NewEvent()
	b.ReportAllocs()
	for b.Loop() {
		e.HasListeners("trace")
	}
}

func BenchmarkEvent_HasListeners_TracingOff_Parallel(b *testing.B) {
	e := NewEvent()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var hits int64
		for pb.Next() {
			if e.HasListeners("trace") {
				hits++
			}
		}
		parallelHits.Add(hits)
	})
}

func BenchmarkEvent_HasListeners_OtherEventRegistered_Parallel(b *testing.B) {
	e := NewEvent()
	e.On("other", func(args ...any) {})
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var hits int64
		for pb.Next() {
			if e.HasListeners("trace") {
				hits++
			}
		}
		parallelHits.Add(hits)
	})
}

func BenchmarkEvent_Emit_OneListener(b *testing.B) {
	e := NewEvent()
	e.On("trace", func(args ...any) {})
	b.ReportAllocs()
	for b.Loop() {
		e.Emit("trace", traceLikeEvent{ID: "abcd1234", Stage: "guard"})
	}
}

func BenchmarkEvent_Emit_OneListener_Parallel(b *testing.B) {
	e := NewEvent()
	e.On("trace", func(args ...any) {})
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		var emitted int64
		for pb.Next() {
			e.Emit("trace", traceLikeEvent{ID: "abcd1234", Stage: "guard"})
			emitted++
		}
		parallelHits.Add(emitted)
	})
}

func BenchmarkEvent_On(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		e := NewEvent()
		e.On("trace", func(args ...any) {})
	}
}
