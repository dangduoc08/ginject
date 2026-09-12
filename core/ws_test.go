package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/trace"
	"golang.org/x/net/websocket"
)

func TestNewWS_DefaultPath(t *testing.T) {
	ws := NewWS(&WSConfig{logger: log.NewLog(nil)})
	if !ws.isWSPath("/ws") {
		t.Error(test.DiffMessage(ws.isWSPath("/ws"), true, "NewWS with no configured Path should default to /ws"))
	}
}

func TestNewWS_CustomPath(t *testing.T) {
	ws := NewWS(&WSConfig{Path: "/live", logger: log.NewLog(nil)})
	if !ws.isWSPath("/live") {
		t.Error(test.DiffMessage(ws.isWSPath("/live"), true, "NewWS should honor a configured custom Path"))
	}
	if ws.isWSPath("/ws") {
		t.Error(test.DiffMessage(ws.isWSPath("/ws"), false, "a custom Path should not also match the default /ws"))
	}
}

func TestIsWSPath_NormalizesSlashes(t *testing.T) {
	ws := NewWS(&WSConfig{Path: "live", logger: log.NewLog(nil)})
	if !ws.isWSPath("/live") {
		t.Error(test.DiffMessage(ws.isWSPath("/live"), true, "isWSPath should normalize a Path configured without leading/trailing slashes"))
	}
}

func TestHandshake_Allowed(t *testing.T) {
	ws := NewWS(&WSConfig{logger: log.NewLog(nil)})

	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	c.ResponseWriter = httptest.NewRecorder()

	if err := ws.handshake(c); err != nil {
		t.Error(test.DiffMessage(err, nil, "handshake with no middlewares should succeed"))
	}
}

func TestHandshake_RejectedWhenMiddlewareDoesNotCallNext(t *testing.T) {
	ws := NewWS(&WSConfig{logger: log.NewLog(nil)})
	ws.globalMiddlewares = []ctx.HTTPHandler{
		func(c *ctx.HTTPContext) {},
	}

	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	c.ResponseWriter = httptest.NewRecorder()

	err := ws.handshake(c)
	if err != errWSHandshakeRejected {
		t.Error(test.DiffMessage(err, errWSHandshakeRejected, "handshake should be rejected when a middleware does not call next()"))
	}
}

type wsHandshakeTraceMiddleware struct{}

func (wsHandshakeTraceMiddleware) Use(_ *http.Request, _ http.ResponseWriter, next ctx.Next) {
	next()
}

func TestHandshake_EmitsMiddlewareTraceWithHTTPTransport(t *testing.T) {
	ev := event.NewEvent()
	var got []trace.Event
	ev.On(trace.EventName, func(args ...any) {
		got = append(got, args[0].(trace.Event))
	})

	ws := NewWS(&WSConfig{
		logger:            log.NewLog(nil),
		event:             ev,
		injectedProviders: map[string]Provider{},
		globalMiddlewares: []common.MiddlewareFn{wsHandshakeTraceMiddleware{}},
	})

	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	c.ResponseWriter = httptest.NewRecorder()

	if err := ws.handshake(c); err != nil {
		t.Fatal(test.DiffMessage(err, nil, "handshake should succeed when the middleware calls next()"))
	}

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 trace event, got %d: %+v", len(got), got)
	}
	if got[0].Stage != trace.StageMiddleware {
		t.Error(test.DiffMessage(got[0].Stage, trace.StageMiddleware, "handshake middleware trace event stage"))
	}
	if got[0].Transport != trace.TransportHTTP {
		t.Error(test.DiffMessage(got[0].Transport, trace.TransportHTTP, "handshake middleware trace event must report HTTP transport, not WS, since the connection has not upgraded yet"))
	}
	if got[0].Name != "core.wsHandshakeTraceMiddleware" {
		t.Error(test.DiffMessage(got[0].Name, "core.wsHandshakeTraceMiddleware", "handshake middleware trace event name"))
	}
}

type wsHandshakeTraceMiddlewareB struct{}

func (wsHandshakeTraceMiddlewareB) Use(_ *http.Request, _ http.ResponseWriter, next ctx.Next) {
	next()
}

func TestHandshake_EmitsOneTraceEventPerMiddleware(t *testing.T) {
	ev := event.NewEvent()
	var got []trace.Event
	ev.On(trace.EventName, func(args ...any) {
		got = append(got, args[0].(trace.Event))
	})

	ws := NewWS(&WSConfig{
		logger:            log.NewLog(nil),
		event:             ev,
		injectedProviders: map[string]Provider{},
		globalMiddlewares: []common.MiddlewareFn{wsHandshakeTraceMiddleware{}, wsHandshakeTraceMiddlewareB{}},
	})

	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	c.ResponseWriter = httptest.NewRecorder()

	if err := ws.handshake(c); err != nil {
		t.Fatal(test.DiffMessage(err, nil, "handshake should succeed when every middleware calls next()"))
	}

	if len(got) != 2 {
		t.Fatalf("expected exactly 2 trace events, got %d: %+v", len(got), got)
	}
	wantNames := []string{"core.wsHandshakeTraceMiddleware", "core.wsHandshakeTraceMiddlewareB"}
	for i, want := range wantNames {
		if got[i].Name != want {
			t.Error(test.DiffMessage(got[i].Name, want, "handshake middleware trace events must fire in registration order"))
		}
		if got[i].Transport != trace.TransportHTTP {
			t.Error(test.DiffMessage(got[i].Transport, trace.TransportHTTP, "every handshake middleware trace event must report HTTP transport"))
		}
	}
}

func TestHandshake_NoListener_StillSucceeds(t *testing.T) {
	ev := event.NewEvent()

	ws := NewWS(&WSConfig{
		logger:            log.NewLog(nil),
		event:             ev,
		injectedProviders: map[string]Provider{},
		globalMiddlewares: []common.MiddlewareFn{wsHandshakeTraceMiddleware{}},
	})

	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/ws", nil)
	c.ResponseWriter = httptest.NewRecorder()

	if err := ws.handshake(c); err != nil {
		t.Fatal(test.DiffMessage(err, nil, "handshake should succeed when the middleware calls next()"))
	}
}

func TestWSHandleRequest_SendsConnectedPayloadAndRegisters(t *testing.T) {
	ws := newTestWSBare(t)
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	done := make(chan struct{})
	go func() {
		ws.handleRequest(serverConn)
		close(done)
	}()

	var got WSPayload
	if err := websocket.JSON.Receive(clientConn, &got); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got.Type != TypeConnected {
		t.Error(test.DiffMessage(got.Type, TypeConnected, "handleRequest should send a connected payload on handshake"))
	}
	if got.ID == "" {
		t.Error(test.DiffMessage(got.ID, "<non-empty connection id>", "connected payload should carry a generated connection id"))
	}

	if _, ok := ws.connmgr.Get(got.ID); !ok {
		t.Error(test.DiffMessage(ok, true, "the connection should be registered under the id sent in the connected payload"))
	}

	_ = clientConn.Close()
	<-done

	if _, ok := ws.connmgr.Get(got.ID); ok {
		t.Error(test.DiffMessage(ok, false, "the connection should be unregistered once handleRequest returns"))
	}
}

func TestHeartbeat_PingLoopStopsWhenDone(t *testing.T) {
	ws := NewWS(&WSConfig{logger: &countingLogger{}, PingInterval: 5 * time.Millisecond})
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		pingLoop(conn, ws.connmgr, done, 5*time.Millisecond)
		close(stopped)
	}()

	time.Sleep(30 * time.Millisecond)
	close(done)

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal(test.DiffMessage("running", "stopped", "the ping goroutine must exit when its connection ends, or every dead connection leaks a goroutine"))
	}
}

func TestBroker_InjectedBrokerReceivesSubscribeAndPublish(t *testing.T) {
	custom := newRecordingBroker()

	ws := NewWS(&WSConfig{logger: &countingLogger{}, Broker: custom})
	ws.newCtx = func() *ctx.WSContext { return ctx.NewWSContext() }
	ws.releaseCtx = func(c *ctx.WSContext) { c.Reset() }
	ws.resolveAndCallHandler = func(any, *ctx.WSContext) []reflect.Value { return nil }
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.1"}})
	drain(t, clientConn, 1)
	handlePublish(conn, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.1"}, Message: "x"})

	subscribed, published := custom.snapshot()
	if len(subscribed) != 1 || subscribed[0] != "chat.1" {
		t.Error(test.DiffMessage(subscribed, []string{"chat.1"}, "subscribe must be delegated to the injected broker"))
	}
	if len(published) != 1 || published[0] != "chat.1" {
		t.Error(test.DiffMessage(published, []string{"chat.1"}, "publish must be delegated to the injected broker"))
	}
}

func TestNewWS_AppliesConfiguredLimits(t *testing.T) {
	shutdown := make(chan struct{})
	defer close(shutdown)

	ws := NewWS(&WSConfig{
		Path:            "ws",
		MaxConnections:  7,
		MaxPayloadBytes: 4096,
		WriteTimeout:    3 * time.Second,
		logger:          log.NewLog(nil),
		shutdownChan:    shutdown,
	})

	if ws.maxPayloadBytes != 4096 {
		t.Error(test.DiffMessage(ws.maxPayloadBytes, 4096, "MaxPayloadBytes must reach the connection"))
	}
	if ws.connmgr.maxConns != 7 {
		t.Error(test.DiffMessage(ws.connmgr.maxConns, 7, "MaxConnections must reach the connection manager"))
	}
	if ws.connmgr.writeTimeout != 3*time.Second {
		t.Error(test.DiffMessage(ws.connmgr.writeTimeout, 3*time.Second, "WriteTimeout must reach the connection manager"))
	}
}

func TestNewWS_UsesSafeDefaultLimits(t *testing.T) {
	shutdown := make(chan struct{})
	defer close(shutdown)

	ws := NewWS(&WSConfig{
		Path:         "ws",
		logger:       log.NewLog(nil),
		shutdownChan: shutdown,
	})

	if ws.maxPayloadBytes != DefaultWSMaxPayloadBytes {
		t.Error(test.DiffMessage(ws.maxPayloadBytes, DefaultWSMaxPayloadBytes, "an unset payload limit must fall back to the default, not to unlimited"))
	}
	if ws.connmgr.maxConns != DefaultWSMaxConnections {
		t.Error(test.DiffMessage(ws.connmgr.maxConns, DefaultWSMaxConnections, "an unset connection limit must fall back to the default"))
	}
	if ws.connmgr.writeTimeout != DefaultWSWriteTimeout {
		t.Error(test.DiffMessage(ws.connmgr.writeTimeout, DefaultWSWriteTimeout, "an unset write timeout must fall back to the default"))
	}
}

func wslHandshakeCtx(t *testing.T, origin string) *ctx.HTTPContext {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/ws", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	c := ctx.NewHTTPContext()
	c.Init(httptest.NewRecorder(), r)

	return c
}

func wslNewWS(t *testing.T, cfg *WSConfig) *WS {
	t.Helper()

	shutdown := make(chan struct{})
	t.Cleanup(func() { close(shutdown) })

	cfg.logger = log.NewLog(nil)
	cfg.shutdownChan = shutdown

	return NewWS(cfg)
}

func TestWSHandshake_AllowedOrigins_RejectsUnlisted(t *testing.T) {
	ws := wslNewWS(t, &WSConfig{
		Path:           "ws",
		AllowedOrigins: []string{"https://app.example.com/"},
	})

	if err := ws.handshake(wslHandshakeCtx(t, "https://app.example.com")); err != nil {
		t.Error(test.DiffMessage(err, nil, "a listed origin must be accepted, trailing slash ignored"))
	}
	if err := ws.handshake(wslHandshakeCtx(t, "https://evil.example")); err != errWSOriginRejected {
		t.Error(test.DiffMessage(err, errWSOriginRejected, "an unlisted origin must be rejected before the upgrade completes"))
	}
}

func TestWSHandshake_AllowedOrigins_AbsentOriginAllowed(t *testing.T) {
	ws := wslNewWS(t, &WSConfig{
		Path:           "ws",
		AllowedOrigins: []string{"https://app.example.com"},
	})

	if err := ws.handshake(wslHandshakeCtx(t, "")); err != nil {
		t.Error(test.DiffMessage(err, nil, "a non-browser client sending no Origin must not be blocked by the allowlist"))
	}
}

func TestWSHandshake_NoAllowedOrigins_AcceptsAnyOrigin(t *testing.T) {
	ws := wslNewWS(t, &WSConfig{Path: "ws"})

	if err := ws.handshake(wslHandshakeCtx(t, "https://evil.example")); err != nil {
		t.Error(test.DiffMessage(err, nil, "without AllowedOrigins the handshake must stay backward compatible"))
	}
}
