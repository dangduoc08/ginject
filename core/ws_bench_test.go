package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/trace"
)

func benchHandshakeContext() *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	c.ResponseWriter = httptest.NewRecorder()
	return c
}

func BenchmarkHandshake_NoListener(b *testing.B) {
	ws := NewWS(&WSConfig{
		logger:            log.NewLog(nil),
		event:             event.NewEvent(),
		injectedProviders: map[string]Provider{},
		globalMiddlewares: []common.MiddlewareFn{wsHandshakeTraceMiddleware{}, wsHandshakeTraceMiddlewareB{}},
	})

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c := benchHandshakeContext()
		_ = ws.handshake(c)
	}
}

func BenchmarkHandshake_WithListener(b *testing.B) {
	ev := event.NewEvent()
	ev.On(trace.EventName, func(args ...any) {})

	ws := NewWS(&WSConfig{
		logger:            log.NewLog(nil),
		event:             ev,
		injectedProviders: map[string]Provider{},
		globalMiddlewares: []common.MiddlewareFn{wsHandshakeTraceMiddleware{}, wsHandshakeTraceMiddlewareB{}},
	})

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c := benchHandshakeContext()
		_ = ws.handshake(c)
	}
}
