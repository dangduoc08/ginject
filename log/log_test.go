package log

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
)

func newTestHandler(level slog.Level) (*PrettyHandler, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	h := NewPrettyHandler(buf, &PrettyHandlerOptions{
		TimeFormat:     time.DateTime,
		HandlerOptions: slog.HandlerOptions{Level: level},
	})
	return h, buf
}

// loadLogOptions tests

func TestLoadLogOptions_NilInput(t *testing.T) {
	opts := loadLogOptions(nil)
	if opts == nil {
		t.Error(test.DiffMessage(nil, "*LogOptions", "nil input should return non-nil opts"))
	}
}

func TestLoadLogOptions_DefaultLogFormat(t *testing.T) {
	opts := loadLogOptions(nil)
	if opts.LogFormat != PrettyFormat {
		t.Error(test.DiffMessage(opts.LogFormat, PrettyFormat, "default LogFormat should be PrettyFormat"))
	}
}

func TestLoadLogOptions_DefaultTimeFormat(t *testing.T) {
	opts := loadLogOptions(nil)
	if opts.TimeFormat != time.DateTime {
		t.Error(test.DiffMessage(opts.TimeFormat, time.DateTime, "default TimeFormat"))
	}
}

func TestLoadLogOptions_PreservesCustomLevel(t *testing.T) {
	opts := loadLogOptions(&LogOptions{Level: ErrorLevel})
	if opts.Level != ErrorLevel {
		t.Error(test.DiffMessage(opts.Level, ErrorLevel, "custom level should be preserved"))
	}
}

func TestLoadLogOptions_PreservesCustomTimeFormat(t *testing.T) {
	opts := loadLogOptions(&LogOptions{TimeFormat: "2006-01-02"})
	if opts.TimeFormat != "2006-01-02" {
		t.Error(test.DiffMessage(opts.TimeFormat, "2006-01-02", "custom TimeFormat"))
	}
}

func TestLoadLogOptions_PreservesCustomLogFormat(t *testing.T) {
	opts := loadLogOptions(&LogOptions{LogFormat: TextFormat})
	if opts.LogFormat != TextFormat {
		t.Error(test.DiffMessage(opts.LogFormat, TextFormat, "custom LogFormat"))
	}
}

func TestLoadLogOptions_LevelMinusOneDefaultsToInfo(t *testing.T) {
	opts := loadLogOptions(&LogOptions{Level: -1})
	if opts.Level != InfoLevel {
		t.Error(test.DiffMessage(opts.Level, InfoLevel, "Level=-1 sentinel should default to InfoLevel"))
	}
}

// NewLog singleton test

func TestNewLog_Singleton(t *testing.T) {
	a := NewLog(nil)
	b := NewLog(&LogOptions{LogFormat: JSONFormat})
	if a != b {
		t.Error(test.DiffMessage(b, a, "NewLog must return the same singleton"))
	}
}

// PrettyHandler.Handle tests

func TestPrettyHandler_Handle_ContainsLevelInfo(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "INFO") {
		t.Error(test.DiffMessage(buf.String(), "contains INFO", "INFO level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelDebug(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelDebug, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "DEBUG") {
		t.Error(test.DiffMessage(buf.String(), "contains DEBUG", "DEBUG level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelWarn(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelWarn, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "WARN") {
		t.Error(test.DiffMessage(buf.String(), "contains WARN", "WARN level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelError(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelError, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "ERROR") {
		t.Error(test.DiffMessage(buf.String(), "contains ERROR", "ERROR level label"))
	}
}

func TestPrettyHandler_Handle_ContainsLevelFatal(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), FatalLevel, "msg", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "FATAL") {
		t.Error(test.DiffMessage(buf.String(), "contains FATAL", "FATAL level label"))
	}
}

func TestPrettyHandler_Handle_ContainsMessage(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "hello world", 0)
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "hello world") {
		t.Error(test.DiffMessage(buf.String(), "contains hello world", "message"))
	}
}

func TestPrettyHandler_Handle_EmptyMessageSkipped(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "", 0)
	_ = h.Handle(context.Background(), r)
	if strings.Contains(buf.String(), "[]") {
		t.Error(test.DiffMessage(buf.String(), "no []", "empty message must not produce [] block"))
	}
}

func TestPrettyHandler_Handle_SingleAttr_UsesLastBranch(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k", "v"))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "└──") {
		t.Error(test.DiffMessage(buf.String(), "contains └──", "single attr should use last-branch symbol"))
	}
}

func TestPrettyHandler_Handle_MultipleAttrs_MiddleAndLastBranch(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k1", "v1"), slog.String("k2", "v2"))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "├──") {
		t.Error(test.DiffMessage(buf.String(), "contains ├──", "first attr should use middle-branch symbol"))
	}
	if !strings.Contains(buf.String(), "└──") {
		t.Error(test.DiffMessage(buf.String(), "contains └──", "last attr should use last-branch symbol"))
	}
}

func TestPrettyHandler_Handle_AttrKeyPresent(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("mykey", "myvalue"))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "mykey") {
		t.Error(test.DiffMessage(buf.String(), "contains mykey", "attr key"))
	}
}

func TestPrettyHandler_Handle_StringAttrValueQuoted(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.String("k", "hello"))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), `"hello"`) {
		t.Error(test.DiffMessage(buf.String(), `contains "hello"`, "string attr value should be quoted"))
	}
}

func TestPrettyHandler_Handle_IntAttrValuePresent(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Int("count", 42))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "42") {
		t.Error(test.DiffMessage(buf.String(), "contains 42", "int attr value"))
	}
}

func TestPrettyHandler_Handle_BoolAttrValuePresent(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Bool("ok", true))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "true") {
		t.Error(test.DiffMessage(buf.String(), "contains true", "bool attr value"))
	}
}

func TestPrettyHandler_Handle_NilAttrShowsNull(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	r.AddAttrs(slog.Any("ptr", nil))
	_ = h.Handle(context.Background(), r)
	if !strings.Contains(buf.String(), "null") {
		t.Error(test.DiffMessage(buf.String(), "contains null", "nil attr value should display as null"))
	}
}

func TestPrettyHandler_Handle_StartsAndEndsWithNewline(t *testing.T) {
	h, buf := newTestHandler(slog.LevelDebug)
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	_ = h.Handle(context.Background(), r)
	out := buf.String()
	if !strings.HasPrefix(out, "\n") {
		t.Error(test.DiffMessage(out, "starts with \\n", "output must start with newline"))
	}
	if !strings.HasSuffix(out, "\n") {
		t.Error(test.DiffMessage(out, "ends with \\n", "output must end with newline"))
	}
}
