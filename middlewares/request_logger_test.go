package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

type mockLogger struct {
	msg    string
	args   []any
	called bool
}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any) {
	m.msg = msg
	m.args = args
	m.called = true
}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}
func (m *mockLogger) Fatal(msg string, args ...any) {}

func findArg(args []any, key string) (any, bool) {
	for i := 0; i+1 < len(args); i += 2 {
		if k, ok := args[i].(string); ok && k == key {
			return args[i+1], true
		}
	}
	return nil, false
}

func newLoggerContext(method, urlPath string, typ string) *ctx.Context {
	req := httptest.NewRequest(method, urlPath, nil)
	rec := httptest.NewRecorder()
	c := ctx.NewContext()
	c.Request = req
	c.ResponseWriter = rec
	c.Event = ctx.NewEvent()
	c.Timestamp = time.Now()
	c.SetType(typ)
	return c
}

func TestRequestLogger_Use_CallsNext(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/", ctx.HTTPType)
	called := false
	rl.Use(c, func() { called = true })
	if !called {
		t.Error(testutils.DiffMessage(called, true, "next should always be called"))
	}
}

func TestRequestLogger_Use_HTTPLogsURL(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/api/users", ctx.HTTPType)
	rl.Use(c, func() {})
	c.Event.Emit(ctx.REQUEST_FINISHED, c)
	if !log.called {
		t.Error(testutils.DiffMessage(log.called, true, "Info should be called"))
		return
	}
	if log.msg != "/api/users" {
		t.Error(testutils.DiffMessage(log.msg, "/api/users", "log message should be URL path"))
	}
}

func TestRequestLogger_Use_HTTPLogsMethod(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodPost, "/api/users", ctx.HTTPType)
	rl.Use(c, func() {})
	c.Event.Emit(ctx.REQUEST_FINISHED, c)
	v, ok := findArg(log.args, "Method")
	if !ok {
		t.Error(testutils.DiffMessage(nil, "Method key", "Method key missing from log args"))
		return
	}
	if v != http.MethodPost {
		t.Error(testutils.DiffMessage(v, http.MethodPost, "Method value mismatch"))
	}
}

func TestRequestLogger_Use_HTTPLogsStatus(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/", ctx.HTTPType)
	c.Code = http.StatusCreated
	rl.Use(c, func() {})
	c.Event.Emit(ctx.REQUEST_FINISHED, c)
	v, ok := findArg(log.args, "Status")
	if !ok {
		t.Error(testutils.DiffMessage(nil, "Status key", "Status key missing from log args"))
		return
	}
	if v != http.StatusCreated {
		t.Error(testutils.DiffMessage(v, http.StatusCreated, "Status value mismatch"))
	}
}

func TestRequestLogger_Use_HTTPLogsProtocol(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/", ctx.HTTPType)
	rl.Use(c, func() {})
	c.Event.Emit(ctx.REQUEST_FINISHED, c)
	v, ok := findArg(log.args, "Protocol")
	if !ok {
		t.Error(testutils.DiffMessage(nil, "Protocol key", "Protocol key missing from log args"))
		return
	}
	if v != "HTTP/1.1" {
		t.Error(testutils.DiffMessage(v, "HTTP/1.1", "Protocol value mismatch"))
	}
}

func TestRequestLogger_Use_HTTPLogsRequestID(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/", ctx.HTTPType)
	c.SetID("req-abc")
	rl.Use(c, func() {})
	c.Event.Emit(ctx.REQUEST_FINISHED, c)
	v, ok := findArg(log.args, ctx.REQUEST_ID)
	if !ok {
		t.Error(testutils.DiffMessage(nil, ctx.REQUEST_ID+" key", "REQUEST_ID key missing from log args"))
		return
	}
	if v != "req-abc" {
		t.Error(testutils.DiffMessage(v, "req-abc", "REQUEST_ID value mismatch"))
	}
}

func TestRequestLogger_Use_HTTPLogsTime(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/", ctx.HTTPType)
	c.Timestamp = time.Now().Add(-50 * time.Millisecond)
	rl.Use(c, func() {})
	c.Event.Emit(ctx.REQUEST_FINISHED, c)
	v, ok := findArg(log.args, "Time")
	if !ok {
		t.Error(testutils.DiffMessage(nil, "Time key", "Time key missing from log args"))
		return
	}
	s, ok := v.(string)
	if !ok || !strings.HasSuffix(s, " ms") {
		t.Error(testutils.DiffMessage(v, "X ms", "Time value should end with ' ms'"))
	}
}

func TestRequestLogger_Use_NoLogWithoutEventEmit(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/", ctx.HTTPType)
	rl.Use(c, func() {})
	if log.called {
		t.Error(testutils.DiffMessage(log.called, false, "Info should not be called before REQUEST_FINISHED"))
	}
}

func TestRequestLogger_Use_UnknownTypeNoLog(t *testing.T) {
	log := &mockLogger{}
	rl := RequestLogger{Logger: log}
	c := newLoggerContext(http.MethodGet, "/", ctx.HTTPType)
	c.Type = ""
	rl.Use(c, func() {})
	c.Event.Emit(ctx.REQUEST_FINISHED, c)
	if log.called {
		t.Error(testutils.DiffMessage(log.called, false, "Info should not be called for unknown type"))
	}
}
