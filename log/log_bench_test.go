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

func BenchmarkPrettyHandler_Handle_NestedStages(b *testing.B) {
	h := NewPrettyHandler(io.Discard, &PrettyHandlerOptions{
		TimeFormat:     time.DateTime,
		HandlerOptions: slog.HandlerOptions{Level: slog.LevelDebug},
	})
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "/api/v1/users", 0)
	r.AddAttrs(
		slog.String("id", "a0eef3e5-b715-4c2e-9c7c-d239e24d305f"),
		slog.String("transport", "HTTP"),
		slog.Int("code", 200),
		slog.String("operation", "GET"),
		slog.Any("middleware", []map[string]Colored{
			{"CORS": {Text: "120µs (4%)", Color: ColorGray}},
			{"CSRF": {Text: "80µs (3%)", Color: ColorGray}},
			{"Helmet": {Text: "40µs (1%)", Color: ColorGray}},
		}),
		slog.Any("guard", []map[string]Colored{
			{"AuthGuard": {Text: "300µs (10%)", Color: ColorGreen}},
		}),
		slog.Any("pre_interceptor", []map[string]Colored{
			{"LogInterceptor": {Text: "15µs (0%)", Color: ColorGray}},
		}),
		slog.Any("handler", []map[string]Colored{
			{"READ_users": {Text: "2ms (68%)", Color: ColorBlue}},
		}),
		slog.Any("post_interceptor", []map[string]Colored{
			{"LogInterceptor": {Text: "90ns (0%)", Color: ColorGray}},
		}),
		slog.String("total", "2.9ms"),
	)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = h.Handle(context.Background(), r)
	}
}

func BenchmarkPrettyHandler_Handle_LargeMap(b *testing.B) {
	h := NewPrettyHandler(io.Discard, &PrettyHandlerOptions{
		TimeFormat:     time.DateTime,
		HandlerOptions: slog.HandlerOptions{Level: slog.LevelDebug},
	})
	meta := make(map[string]int, 20)
	for i := range 20 {
		meta[string(rune('a'+i%26))+string(rune('A'+i))] = i
	}
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "meta", 0)
	r.AddAttrs(slog.Any("meta", meta))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = h.Handle(context.Background(), r)
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

type benchTagPerson struct {
	Age    int    `log:"age"`
	name   string `log:"name"`
	School string
	policy string
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}
func (noopLogger) Fatal(string, ...any) {}

func BenchmarkWrapLogger_Info_TaggedStruct(b *testing.B) {
	l := WrapLogger(noopLogger{}, nil)
	p := benchTagPerson{Age: 10, name: "Joh", School: "MIT", policy: "ok"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		l.Info("test", "person", p)
	}
}

func BenchmarkWrapLogger_Info_TaggedStruct_WithMaskRule(b *testing.B) {
	l := WrapLogger(noopLogger{}, []string{"person.name"})
	p := benchTagPerson{Age: 10, name: "Joh", School: "MIT", policy: "ok"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		l.Info("test", "person", p)
	}
}

func BenchmarkWrapLogger_Info_NoStructArgs(b *testing.B) {
	l := WrapLogger(noopLogger{}, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		l.Info("test", "url", "/api/v1/users", "status", 200)
	}
}

func BenchmarkJSONHandler_Handle_NoStructAttrs_Unwrapped(b *testing.B) {
	h := slog.NewJSONHandler(io.Discard, nil)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("url", "/api/v1/users"), slog.Int("status", 200))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_ = h.Handle(context.Background(), r)
	}
}
