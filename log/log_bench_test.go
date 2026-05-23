package log

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func BenchmarkPrettyHandler_Handle_NoAttrs(b *testing.B) {
	h := NewPrettyHandler(io.Discard, &PrettyHandlerOptions{
		TimeFormat:     time.DateTime,
		HandlerOptions: slog.HandlerOptions{Level: slog.LevelDebug},
	})
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
	b.ResetTimer()
	for range b.N {
		_ = h.Handle(context.Background(), r)
	}
}

func BenchmarkPrettyHandler_Handle_WithAttrs(b *testing.B) {
	h := NewPrettyHandler(io.Discard, &PrettyHandlerOptions{
		TimeFormat:     time.DateTime,
		HandlerOptions: slog.HandlerOptions{Level: slog.LevelDebug},
	})
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
	r.AddAttrs(
		slog.String("url", "/api/v1/users"),
		slog.Int("status", 200),
		slog.Bool("cached", false),
	)
	b.ResetTimer()
	for range b.N {
		_ = h.Handle(context.Background(), r)
	}
}

func BenchmarkLoadLogOptions_Default(b *testing.B) {
	for range b.N {
		loadLogOptions(nil)
	}
}

func BenchmarkLoadLogOptions_Custom(b *testing.B) {
	opts := &LogOptions{
		Level:      ErrorLevel,
		LogFormat:  TextFormat,
		TimeFormat: "2006-01-02",
	}
	for range b.N {
		loadLogOptions(opts)
	}
}

func BenchmarkNewPrettyHandler(b *testing.B) {
	for range b.N {
		NewPrettyHandler(io.Discard, &PrettyHandlerOptions{
			TimeFormat:     time.DateTime,
			HandlerOptions: slog.HandlerOptions{Level: slog.LevelDebug},
		})
	}
}
