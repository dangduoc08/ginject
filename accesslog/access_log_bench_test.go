package accesslog

import (
	"testing"
	"time"

	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/trace"
)

func benchEntries() []trace.Event {
	return []trace.Event{
		{ID: "req-1", Stage: trace.StageMiddleware, Name: "cors.CORS", Duration: 120 * time.Microsecond},
		{ID: "req-1", Stage: trace.StageMiddleware, Name: "csrf.CSRF", Duration: 80 * time.Microsecond},
		{ID: "req-1", Stage: trace.StageMiddleware, Name: "helmet.Helmet", Duration: 40 * time.Microsecond},
		{ID: "req-1", Stage: trace.StageGuard, Name: "auth.AuthGuard", Duration: 300 * time.Microsecond},
		{ID: "req-1", Stage: trace.StagePreInterceptor, Name: "log.LogInterceptor", Duration: 15 * time.Microsecond},
		{ID: "req-1", Stage: trace.StageHandler, Name: "READ_users", Duration: 2 * time.Millisecond},
	}
}

func BenchmarkStageArgs(b *testing.B) {
	entries := benchEntries()
	total := 3 * time.Millisecond
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		stageArgs(entries, total)
	}
}

func BenchmarkDistributePercentages(b *testing.B) {
	entries := benchEntries()
	total := sumDurations(entries)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		distributePercentages(entries, total)
	}
}

func BenchmarkFormatDuration(b *testing.B) {
	durations := []time.Duration{
		500 * time.Nanosecond,
		1234 * time.Nanosecond,
		1234567 * time.Nanosecond,
		1234567891 * time.Nanosecond,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatDuration(durations[i%len(durations)])
	}
}

func BenchmarkColorForDuration(b *testing.B) {
	durations := []time.Duration{
		5 * time.Millisecond,
		40 * time.Millisecond,
		90 * time.Millisecond,
		250 * time.Millisecond,
		700 * time.Millisecond,
		2 * time.Second,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		colorForDuration(durations[i%len(durations)])
	}
}

func BenchmarkShortName(b *testing.B) {
	names := []string{"cors.CORS", "csrf.CSRF", "READ_users", "pkg.sub.Type"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shortName(names[i%len(names)])
	}
}

func BenchmarkAccessLog_EmitThroughput(b *testing.B) {
	logger := &fakeLogger{ch: make(chan recordedLog, 1024)}
	ev := event.NewEvent()
	NewAccessLog(&AccessLogConfig{Event: ev, Logger: logger})

	entries := benchEntries()
	done := trace.Event{ID: "req-1", Stage: trace.StageComplete, Code: 200, Target: "Request", Duration: 3 * time.Millisecond}

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-logger.ch:
			case <-stop:
				return
			}
		}
	}()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		for _, e := range entries {
			ev.Emit(trace.EventName, e)
		}
		ev.Emit(trace.EventName, done)
	}
	b.StopTimer()
	close(stop)
}
