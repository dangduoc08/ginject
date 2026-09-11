package event

import "testing"

func BenchmarkEvent_On(b *testing.B) {
	e := NewEvent()
	e.SetMaxListeners(0)
	fn := func(args ...any) {}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.On("x", fn)
	}
}

func BenchmarkEvent_EmitNoListeners(b *testing.B) {
	e := NewEvent()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Emit("x", "payload")
	}
}

func BenchmarkEvent_EmitWithListeners(b *testing.B) {
	e := NewEvent()
	e.SetMaxListeners(0)
	for i := 0; i < 10; i++ {
		e.On("x", func(args ...any) {})
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Emit("x", "payload")
	}
}

func BenchmarkEvent_ListenerCount(b *testing.B) {
	e := NewEvent()
	e.On("x", func(args ...any) {})
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ListenerCount("x")
	}
}

func BenchmarkEmit_Parallel_NoOnceListeners(b *testing.B) {
	e := NewEvent()
	e.On("trace", func(args ...any) {})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			e.Emit("trace", 1)
		}
	})
}

func BenchmarkHasListeners_Parallel(b *testing.B) {
	e := NewEvent()
	e.On("trace", func(args ...any) {})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			e.HasListeners("trace")
		}
	})
}
