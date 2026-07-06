package requestlogger

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"golang.org/x/net/websocket"
)

type noopLogger struct{}

func (noopLogger) Debug(msg string, args ...any) {}
func (noopLogger) Info(msg string, args ...any)  {}
func (noopLogger) Warn(msg string, args ...any)  {}
func (noopLogger) Error(msg string, args ...any) {}
func (noopLogger) Fatal(msg string, args ...any) {}

func BenchmarkRequestLogger_Use_HTTP(b *testing.B) {
	rl := RequestLogger{Logger: noopLogger{}}
	b.ResetTimer()
	for range b.N {
		c := newLoggerContext(http.MethodGet, "/api/users", ctx.HTTPType)
		c.Timestamp = time.Now().Add(-10 * time.Millisecond)
		rl.Use(c, func() {})
		_ = c.Broker.Publish(ctx.RequestFinished, c)
	}
}

func BenchmarkRequestLogger_Use_RegisterOnly(b *testing.B) {
	rl := RequestLogger{Logger: noopLogger{}}
	c := newLoggerContext(http.MethodGet, "/api/users", ctx.HTTPType)
	b.ResetTimer()
	for range b.N {
		rl.Use(c, func() {})
	}
}

func BenchmarkRequestLogger_Use_WS(b *testing.B) {
	rl := RequestLogger{Logger: noopLogger{}}
	b.ResetTimer()
	for range b.N {
		c := newLoggerContext(http.MethodGet, "/ws", ctx.WSType)
		c.SetWSConfig(&websocket.Config{Location: &url.URL{Path: "/chat"}})
		c.Timestamp = time.Now().Add(-10 * time.Millisecond)
		rl.Use(c, func() {})
		_ = c.Broker.Publish(ctx.RequestFinished, c)
	}
}
