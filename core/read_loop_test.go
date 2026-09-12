package core

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/memorybroker"
	"github.com/dangduoc08/ginject/trace"
)

// newTestWSBare builds a *WS with newCtx/releaseCtx/resolveAndCallHandler
// wired the same way app.go's Create does — dispatchWSEvent needs all three
// to run a PUBLISH through its Middlewares/Handler — but no event patterns
// pre-registered, so callers can register their own via
// ws.eventMatcher.AddInjectableHandler/AddMiddlewares for scenarios that
// need a specific handler or middleware.
func newTestWSBare(t testing.TB) *WS {
	t.Helper()

	ws := NewWS(&WSConfig{logger: log.NewLog(nil)})
	ws.newCtx = func() *ctx.WSContext {
		c := ctx.NewWSContext()
		return c
	}
	ws.releaseCtx = func(c *ctx.WSContext) {
		c.Reset()
	}
	ev := event.NewEvent()
	ws.resolveAndCallHandler = func(f any, c *ctx.WSContext) []reflect.Value {
		var pipeElapsed time.Duration
		var handlerCalled bool
		return invokeWSHandlerByProviders(f, nil, c, ev, &pipeElapsed, &handlerCalled)
	}
	ws.emitPostInterceptor = func(c *ctx.WSContext, name string, duration time.Duration) {}
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {}

	return ws
}

// newTestWS builds a *WS with eventMatcher pre-populated as if the given
// event patterns had been registered by real SUBSCRIBE_xxx controllers,
// using a no-op handler. Only the pattern's presence matters for the
// whitelist-only tests that use this helper.
func newTestWS(t testing.TB, eventPatterns ...string) *WS {
	t.Helper()

	ws := newTestWSBare(t)
	for _, p := range eventPatterns {
		ws.eventMatcher.AddInjectableHandler(p, func() {})
	}

	return ws
}

func recvWSPayload(t testing.TB, conn *websocket.Conn) WSPayload {
	t.Helper()

	var p WSPayload
	if err := websocket.JSON.Receive(conn, &p); err != nil {
		t.Fatalf("receive: %v", err)
	}
	return p
}

func TestHandleSubscribe_WhitelistRejectsUnknownTopic(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"random.topic"}})

	var got WSPayload
	if err := websocket.JSON.Receive(clientConn, &got); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got.Type != TypeError {
		t.Error(test.DiffMessage(got.Type, TypeError, "subscribing to a topic with no matching SUBSCRIBE_xxx pattern should be rejected"))
	}
}

func TestHandleSubscribe_WhitelistAcceptsMatchingTopic(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	if !ws.connmgr.isSubscribed(conn.ID, "chat.to.user2") {
		t.Error(test.DiffMessage(false, true, "subscribing to a topic matching a registered pattern should succeed"))
	}
}

func TestHandlePublish_RejectsWithoutPriorSubscribe(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handlePublish(conn, ws, WSPayload{ID: "req-1", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

	var got WSPayload
	if err := websocket.JSON.Receive(clientConn, &got); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got.Type != TypeError {
		t.Error(test.DiffMessage(got.Type, TypeError, "publish before subscribe should be rejected (protocol validity, not a Guard concern)"))
	}
}

func TestHandlePublish_RejectsUnknownTopic(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handlePublish(conn, ws, WSPayload{ID: "req-1", Type: TypePublish, Topic: []string{"random.topic"}, Message: "hi"})

	var got WSPayload
	if err := websocket.JSON.Receive(clientConn, &got); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got.Type != TypeError {
		t.Error(test.DiffMessage(got.Type, TypeError, "publish to a topic with no matching SUBSCRIBE_xxx pattern should be rejected"))
	}
}

func TestHandlePublish_DeliversAfterSubscribe(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	// Drain subscribe ACK
	subscribeAck := recvWSPayload(t, clientConn)
	if subscribeAck.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", subscribeAck.Type)
	}

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

	// Receive broker fan-out event
	var p WSPayload
	if err := websocket.JSON.Receive(clientConn, &p); err != nil {
		t.Fatalf("receive: %v", err)
	}

	if p.Type != TypeEvent {
		t.Error(test.DiffMessage(p.Type, TypeEvent, "expected the event the broker fans back out to this same connection"))
	}
	if len(p.Topic) != 1 || p.Topic[0] != "chat.to.user2" || p.Message != "hi" {
		t.Error(test.DiffMessage(p, "event chat.to.user2 hi", "unexpected event payload delivered via broker"))
	}

	// Receive publish ACK
	ack := recvWSPayload(t, clientConn)
	if ack.Type != TypeAck {
		t.Errorf("expected publish ack, got %v", ack.Type)
	}
}

func TestDispatchWSEvent_HandlerReturnValueRepliesAsTypeResponse(t *testing.T) {
	ws := newTestWSBare(t)
	ws.eventMatcher.AddInjectableHandler("chat.to.*", func() ctx.Map {
		return ctx.Map{"reply": "ack-from-handler"}
	})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	// Drain subscribe ACK
	subscribeAck := recvWSPayload(t, clientConn)
	if subscribeAck.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", subscribeAck.Type)
	}

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

	handlerReplyFrame := recvWSPayload(t, clientConn)
	fanOutFrame := recvWSPayload(t, clientConn)

	if handlerReplyFrame.Type != TypeResponse {
		t.Fatalf("expected first frame to be the handler's TypeResponse reply, got %v", handlerReplyFrame.Type)
	}
	if len(handlerReplyFrame.Topic) != 1 || handlerReplyFrame.Topic[0] != "chat.to.user2" {
		t.Error(test.DiffMessage(handlerReplyFrame.Topic, []string{"chat.to.user2"}, "handler response must carry the topic it answers so the client can route it"))
	}
	handlerReply, ok := handlerReplyFrame.Message.(map[string]any)
	if !ok || handlerReply["reply"] != "ack-from-handler" {
		t.Error(test.DiffMessage(handlerReplyFrame.Message, map[string]any{"reply": "ack-from-handler"}, "handler's return value should be sent back via TypeResponse"))
	}

	if fanOutFrame.Type != TypeEvent || fanOutFrame.Message != "hi" {
		t.Error(test.DiffMessage(fanOutFrame, "TypeEvent hi", "broker fan-out should still deliver the original published message"))
	}
}

// Guards run on subscribe, unsubscribe and publish (only the global
// handshake-time middlewares are exempt), so a Guard that unconditionally
// denies blocks subscribe itself — there's no way to reach publish's
// must-already-be-subscribed check at all.
func TestHandleSubscribe_GuardDenialBlocksSubscribeAndRepliesError(t *testing.T) {
	ws := newTestWSBare(t)
	ws.eventMatcher.AddMiddlewares("chat.to.*", common.BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return false }))
	ws.eventMatcher.AddInjectableHandler("chat.to.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	got := recvWSPayload(t, clientConn)
	if got.Type != TypeError {
		t.Fatalf("expected a denied Guard to reject subscribe with TypeError, got %v", got.Type)
	}

	if ws.connmgr.isSubscribed(conn.ID, "chat.to.user2") {
		t.Error("connection should not be registered as subscribed after a Guard denial")
	}

	// No further frames (no ack) should follow a Guard rejection.
	if err := clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	var extra WSPayload
	if err := websocket.JSON.Receive(clientConn, &extra); err == nil {
		t.Errorf("expected no further frames after a Guard rejection, got %+v", extra)
	}
}

func TestDispatchWSEvent_GuardDenialBlocksFanOutAndRepliesError(t *testing.T) {
	ws := newTestWSBare(t)
	ws.eventMatcher.AddInjectableHandler("chat.to.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	// Drain subscribe ACK
	subscribeAck := recvWSPayload(t, clientConn)
	if subscribeAck.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", subscribeAck.Type)
	}

	ws.eventMatcher.AddMiddlewares("chat.to.*", common.BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return false }))

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

	got := recvWSPayload(t, clientConn)
	if got.Type != TypeError {
		t.Fatalf("expected a denied Guard to reply TypeError, got %v", got.Type)
	}

	// No further frames (no fan-out event, no publish ack) should follow a
	// Guard rejection.
	if err := clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline: %v", err)
	}
	var extra WSPayload
	if err := websocket.JSON.Receive(clientConn, &extra); err == nil {
		t.Errorf("expected no further frames after a Guard rejection, got %+v", extra)
	}
}

func TestDispatchWSEvent_HandlerPanicRepliesErrorAndConnectionSurvives(t *testing.T) {
	ws := newTestWSBare(t)
	ws.eventMatcher.AddInjectableHandler("chat.to.*", func() ctx.Map {
		panic("boom")
	})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	// Drain subscribe ACK
	subscribeAck := recvWSPayload(t, clientConn)
	if subscribeAck.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", subscribeAck.Type)
	}

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

	got := recvWSPayload(t, clientConn)
	if got.Type != TypeError {
		t.Fatalf("expected a panicking handler to reply TypeError instead of crashing the caller, got %v", got.Type)
	}

	// A second publish on the same connection should still work normally,
	// proving the panic didn't corrupt ws/conn state.
	handlePublish(conn, ws, WSPayload{ID: "req-3", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi again"})
	got2 := recvWSPayload(t, clientConn)
	if got2.Type != TypeError {
		t.Fatalf("expected second publish to also reply TypeError (same panicking handler), got %v", got2.Type)
	}
}

func TestDispatchWSEvent_InjectsWSPayloadIntoHandler(t *testing.T) {
	ws := newTestWSBare(t)

	var got ctx.WSPayload
	ws.eventMatcher.AddInjectableHandler("chat.to.*", func(p ctx.WSPayload) {
		got = p
	})

	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})
	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: map[string]any{"foo": "bar"}})

	if got == nil || got["foo"] != "bar" {
		t.Error(test.DiffMessage(got, ctx.WSPayload{"foo": "bar"}, "handler should receive the published message as ctx.WSPayload"))
	}
}

func TestReadLoop_DispatchesSubscribeAndUnsupportedType(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	done := make(chan struct{})
	go func() {
		readLoop(conn, ws)
		close(done)
	}()

	if err := websocket.JSON.Send(clientConn, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}}); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)
	if !ws.connmgr.isSubscribed(conn.ID, "chat.to.user2") {
		t.Error(test.DiffMessage(false, true, "readLoop should dispatch a subscribe message to handleSubscribe"))
	}

	// Drain subscribe ACK
	subscribeAck := recvWSPayload(t, clientConn)
	if subscribeAck.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", subscribeAck.Type)
	}

	if err := websocket.JSON.Send(clientConn, WSPayload{ID: "req-2", Type: "bogus"}); err != nil {
		t.Fatal(err)
	}
	got := recvWSPayload(t, clientConn)
	if got.Type != TypeError {
		t.Error(test.DiffMessage(got.Type, TypeError, "readLoop should reply with an error for an unsupported payload type"))
	}

	_ = clientConn.Close()
	<-done
}

func TestHandleUnsubscribe_RemovesTopicAndAcks(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	handleUnsubscribe(conn, ws, WSPayload{ID: "req-2", Topic: []string{"chat.to.user2"}})

	if ws.connmgr.isSubscribed("conn-1", "chat.to.user2") {
		t.Error(test.DiffMessage(true, false, "handleUnsubscribe should remove the subscription"))
	}
}

func TestDispatchWSEvent_GuardEmitsTraceWithWSTransport(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageGuard {
			d := te
			got = &d
		}
	})

	pattern := "chat.to.*"
	mw := traceWSHandler(ev, trace.StageGuard, "test.AllowGuard", common.BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return true }))
	ws.eventMatcher.AddMiddlewares(pattern, mw)
	ws.eventMatcher.AddInjectableHandler(pattern, func() {})

	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	mu.Lock()
	defer mu.Unlock()
	if got == nil {
		t.Fatal("expected a guard trace event")
	}
	if got.Stage != trace.StageGuard || got.Name != "test.AllowGuard" || got.Transport != trace.TransportWS {
		t.Error(test.DiffMessage([]any{got.Stage, got.Name, got.Transport}, []any{trace.StageGuard, "test.AllowGuard", trace.TransportWS}, "WS guard trace event fields"))
	}
}

func TestDispatchWSEvent_ExceptionFilterEmitsTraceWithWSTransport(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageExceptionFilter {
			d := te
			got = &d
		}
	})

	pattern := "chat.to.*"
	ws.eventMatcher.AddInjectableHandler(pattern, func() { panic("boom") })
	catchFn := traceWSCatch(ev, "test.Filter", func(c *ctx.WSContext, ex *exception.Exception) {
		c.Send(ctx.Map{"caught": true})
	})
	ws.catchFnsByEvent[pattern] = append(ws.catchFnsByEvent[pattern], catchFn)

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	defer mu.Unlock()
	if got == nil {
		t.Fatal("expected an exceptionFilter trace event")
	}
	if got.Stage != trace.StageExceptionFilter || got.Name != "test.Filter" || got.Transport != trace.TransportWS {
		t.Error(test.DiffMessage([]any{got.Stage, got.Name, got.Transport}, []any{trace.StageExceptionFilter, "test.Filter", trace.TransportWS}, "WS exceptionFilter trace event fields"))
	}
}

func TestDispatchWSEvent_InterceptorPrePostEventsMatchInCountAndOrder(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var pre, post []string
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StagePreInterceptor:
			pre = append(pre, te.Name)
		case trace.StagePostInterceptor:
			post = append(post, te.Name)
		}
	})
	ws.emitPostInterceptor = func(c *ctx.WSContext, name string, duration time.Duration) {
		ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StagePostInterceptor, Name: name, Transport: trace.TransportWS, Duration: duration})
	}

	pattern := "chat.to.*"
	pipeIntercept := func(_ *ctx.WSContext, agg *aggregation.Aggregation) any { return agg.Pipe() }
	for _, name := range []string{"A", "B", "C"} {
		mw := traceWSHandler(ev, trace.StagePreInterceptor, name, common.BuildWSInterceptMiddleware(pattern, pipeIntercept))
		mw = tagWSInterceptorName(ev, pattern, name, mw)
		ws.eventMatcher.AddMiddlewares(pattern, mw)
	}
	ws.eventMatcher.AddInjectableHandler(pattern, func() string { return "ok" })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	mu.Lock()
	pre = nil
	post = nil
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	defer mu.Unlock()
	if len(pre) != 3 {
		t.Fatalf("expected 3 pre-interceptor events, got %d: %v", len(pre), pre)
	}
	if len(post) != 3 {
		t.Fatalf("expected 3 post-interceptor events, got %d: %v", len(post), post)
	}
	wantPost := []string{"C", "B", "A"}
	for i, want := range wantPost {
		if post[i] != want {
			t.Error(test.DiffMessage(post, wantPost, "post-interceptor events must fire in reverse execution order of pre-interceptor"))
			break
		}
	}
}

func TestDispatchWSEvent_InterceptorPrePostEventsMatchWhenShortCircuited(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var preCount, postCount int
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StagePreInterceptor:
			preCount++
		case trace.StagePostInterceptor:
			postCount++
		}
	})
	ws.emitPostInterceptor = func(c *ctx.WSContext, name string, duration time.Duration) {
		ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StagePostInterceptor, Name: name, Transport: trace.TransportWS, Duration: duration})
	}

	pattern := "chat.to.*"
	pipeIntercept := func(_ *ctx.WSContext, agg *aggregation.Aggregation) any { return agg.Pipe() }
	shortCircuitIntercept := func(_ *ctx.WSContext, agg *aggregation.Aggregation) any { return "short-circuited" }

	for _, spec := range []struct {
		name string
		fn   common.WSIntercept
	}{
		{"A", pipeIntercept},
		{"ShortCircuit", shortCircuitIntercept},
		{"C", pipeIntercept},
	} {
		mw := traceWSHandler(ev, trace.StagePreInterceptor, spec.name, common.BuildWSInterceptMiddleware(pattern, spec.fn))
		mw = tagWSInterceptorName(ev, pattern, spec.name, mw)
		ws.eventMatcher.AddMiddlewares(pattern, mw)
	}
	ws.eventMatcher.AddInjectableHandler(pattern, func() string { return "ok" })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	mu.Lock()
	preCount = 0
	postCount = 0
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	defer mu.Unlock()
	if preCount != 3 {
		t.Fatalf("expected 3 pre-interceptor events, got %d", preCount)
	}
	if postCount != preCount {
		t.Error(test.DiffMessage(postCount, preCount, "post-interceptor event count must match pre-interceptor count even when an interceptor short-circuits"))
	}
}

func TestDispatchWSEvent_EmitsCompleteWithWSTransport(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageComplete {
			d := te
			got = &d
		}
	})
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {
		ev.Emit(trace.EventName, trace.Event{
			ID:        c.GetID(),
			Stage:     trace.StageComplete,
			Transport: trace.TransportWS,
			Operation: operation,
			Target:    target,
			Duration:  time.Since(c.Timestamp),
		})
	}

	pattern := "chat.to.*"
	ws.eventMatcher.AddInjectableHandler(pattern, func() string { return "ok" })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	mu.Lock()
	got = nil
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	defer mu.Unlock()
	if got == nil {
		t.Fatal("expected a StageComplete trace event for the publish dispatch")
	}
	if got.Transport != trace.TransportWS {
		t.Error(test.DiffMessage(got.Transport, trace.TransportWS, "WS dispatch StageComplete must report WS transport"))
	}
	if got.Operation != string(TypePublish) {
		t.Error(test.DiffMessage(got.Operation, string(TypePublish), "StageComplete Operation should be the payload type"))
	}
	if got.Target != "chat.to.user2" {
		t.Error(test.DiffMessage(got.Target, "chat.to.user2", "StageComplete Target should be the concrete topic"))
	}
	if got.ID == "" {
		t.Error(test.DiffMessage(got.ID, "<non-empty>", "StageComplete must carry a non-empty request id"))
	}
}

func TestDispatchWSEvent_CompleteIDMatchesGuardEventID(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var guardID, completeID string
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StageGuard:
			guardID = te.ID
		case trace.StageComplete:
			completeID = te.ID
		}
	})
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {
		ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StageComplete, Transport: trace.TransportWS, Operation: operation, Target: target})
	}

	pattern := "chat.to.*"
	mw := traceWSHandler(ev, trace.StageGuard, "test.AllowGuard", common.BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return true }))
	ws.eventMatcher.AddMiddlewares(pattern, mw)
	ws.eventMatcher.AddInjectableHandler(pattern, func() string { return "ok" })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	mu.Lock()
	guardID, completeID = "", ""
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	defer mu.Unlock()
	if guardID == "" || completeID == "" {
		t.Fatalf("expected both a guard event and a complete event, got guardID=%q completeID=%q", guardID, completeID)
	}
	if guardID != completeID {
		t.Error(test.DiffMessage(completeID, guardID, "StageComplete must carry the same id as the guard event from the same dispatch, so accesslog can correlate and flush them together"))
	}
}

func TestDispatchWSEvent_CompleteFiresOnGuardDenial(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var completeCount int
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageComplete {
			completeCount++
		}
	})
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {
		ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StageComplete, Transport: trace.TransportWS, Operation: operation, Target: target})
	}

	pattern := "chat.to.*"
	ws.eventMatcher.AddInjectableHandler(pattern, func() string { return "ok" })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	ws.eventMatcher.AddMiddlewares(pattern, common.BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return false }))

	mu.Lock()
	completeCount = 0
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	defer mu.Unlock()
	if completeCount != 1 {
		t.Error(test.DiffMessage(completeCount, 1, "StageComplete must still fire exactly once when a guard denies the dispatch"))
	}
}

func TestDispatchWSEvent_CompleteFiresOnHandlerPanic(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var completeCount int
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageComplete {
			completeCount++
		}
	})
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {
		ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StageComplete, Transport: trace.TransportWS, Operation: operation, Target: target})
	}

	pattern := "chat.to.*"
	ws.eventMatcher.AddInjectableHandler(pattern, func() { panic("boom") })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	mu.Lock()
	completeCount = 0
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	defer mu.Unlock()
	if completeCount != 1 {
		t.Error(test.DiffMessage(completeCount, 1, "StageComplete must still fire exactly once when the handler panics"))
	}
}

func TestHandleSubscribe_EmitsCompleteWithWSTransport(t *testing.T) {
	ws := newTestWSBare(t)
	ev := event.NewEvent()
	var mu sync.Mutex
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageComplete {
			d := te
			got = &d
		}
	})
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {
		ev.Emit(trace.EventName, trace.Event{
			ID:        c.GetID(),
			Stage:     trace.StageComplete,
			Transport: trace.TransportWS,
			Operation: operation,
			Target:    target,
			Duration:  time.Since(c.Timestamp),
		})
	}

	pattern := "chat.to.*"
	ws.eventMatcher.AddInjectableHandler(pattern, func() string { return "ok" })

	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	mu.Lock()
	defer mu.Unlock()
	if got == nil {
		t.Fatal("expected a StageComplete trace event for the subscribe dispatch")
	}
	if got.Transport != trace.TransportWS {
		t.Error(test.DiffMessage(got.Transport, trace.TransportWS, "WS dispatch StageComplete must report WS transport"))
	}
	if got.Operation != string(TypeSubscribe) {
		t.Error(test.DiffMessage(got.Operation, string(TypeSubscribe), "StageComplete Operation should be the payload type"))
	}
	if got.Target != "chat.to.user2" {
		t.Error(test.DiffMessage(got.Target, "chat.to.user2", "StageComplete Target should be the concrete topic"))
	}
}

func TestReadLoop_RespondsToClientPing(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	done := make(chan struct{})
	go func() {
		readLoop(conn, ws)
		close(done)
	}()

	if err := websocket.JSON.Send(clientConn, WSPayload{Type: TypePing}); err != nil {
		t.Fatal(err)
	}

	pong := recvWSPayload(t, clientConn)
	if pong.Type != TypePong {
		t.Fatalf("expected PONG response to PING, got %v", pong.Type)
	}

	_ = clientConn.Close()
	<-done
}

type traceRecorder struct {
	mu     sync.Mutex
	ev     *event.Event
	events []trace.Event
}

func newTraceRecorder() *traceRecorder {
	r := &traceRecorder{ev: event.NewEvent()}
	r.ev.On(trace.EventName, func(args ...any) {
		r.mu.Lock()
		defer r.mu.Unlock()
		r.events = append(r.events, args[0].(trace.Event))
	})
	return r
}

func (r *traceRecorder) snapshot() []trace.Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]trace.Event, len(r.events))
	copy(out, r.events)
	return out
}

func (r *traceRecorder) byStage(stage string) []trace.Event {
	var out []trace.Event
	for _, e := range r.snapshot() {
		if e.Stage == stage {
			out = append(out, e)
		}
	}
	return out
}

func (r *traceRecorder) completeFor(operation, target string) *trace.Event {
	for _, e := range r.snapshot() {
		if e.Stage == trace.StageComplete && e.Operation == operation && e.Target == target {
			d := e
			return &d
		}
	}
	return nil
}

// newTracedWS wires a *WS the way app.Create does, but against a recorder so a
// test can assert on the exact trace events an operation produces.
func newTracedWS(t testing.TB, rec *traceRecorder) *WS {
	t.Helper()

	ws := NewWS(&WSConfig{logger: log.NewLog(nil)})
	ws.newCtx = func() *ctx.WSContext { return ctx.NewWSContext() }
	ws.releaseCtx = func(c *ctx.WSContext) { c.Reset() }
	ws.resolveAndCallHandler = func(f any, c *ctx.WSContext) []reflect.Value {
		start := time.Now()
		var pipeElapsed time.Duration
		var handlerCalled bool
		defer func() {
			if !handlerCalled {
				return
			}
			ws.emitHandler(c, "handler:"+c.Pattern(), time.Since(start)-pipeElapsed)
		}()
		return invokeWSHandlerByProviders(f, nil, c, rec.ev, &pipeElapsed, &handlerCalled)
	}
	ws.emitPostInterceptor = func(c *ctx.WSContext, name string, duration time.Duration) {
		rec.ev.Emit(trace.EventName, trace.Event{
			ID: c.GetID(), ConnID: c.ConnID(), Stage: trace.StagePostInterceptor,
			Name: name, Transport: trace.TransportWS, Duration: duration,
		})
	}
	ws.emitHandler = func(c *ctx.WSContext, name string, duration time.Duration) {
		rec.ev.Emit(trace.EventName, trace.Event{
			ID: c.GetID(), ConnID: c.ConnID(), Stage: trace.StageHandler,
			Name: name, Transport: trace.TransportWS, Duration: duration,
		})
	}
	ws.emitComplete = func(c *ctx.WSContext, operation, target, status string, code int) {
		rec.ev.Emit(trace.EventName, trace.Event{
			ID: c.GetID(), ConnID: c.ConnID(), Stage: trace.StageComplete,
			Transport: trace.TransportWS, Operation: operation, Target: target,
			Status: status, Code: code, Duration: time.Since(c.Timestamp),
		})
	}

	return ws
}

func drain(t testing.TB, clientConn *websocket.Conn, n int) []WSPayload {
	t.Helper()

	out := make([]WSPayload, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, recvWSPayload(t, clientConn))
	}
	return out
}

func resultsOf(t testing.TB, p WSPayload) []map[string]any {
	t.Helper()

	raw, ok := p.Message.([]any)
	if !ok {
		t.Fatalf("expected a per-topic result array, got %T (%v)", p.Message, p.Message)
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("expected a result object, got %T", item)
		}
		out = append(out, m)
	}
	return out
}

func TestSubscribe_GuardSeesTopicConnIDAndPattern(t *testing.T) {
	ws := newTestWSBare(t)

	var gotTopic, gotPattern, gotConnID, gotOperation string
	ws.eventMatcher.AddGuards("chat.person.*", common.BuildWSGuardMiddleware(func(c *ctx.WSContext) bool {
		gotTopic = c.Topic()
		gotPattern = c.Pattern()
		gotConnID = c.ConnID()
		gotOperation = c.Operation()
		return true
	}))
	ws.eventMatcher.AddInjectableHandler("chat.person.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-guard", serverConn)
	defer ws.connmgr.Unregister("conn-guard")

	handleSubscribe(conn, ws, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.person.42"}})
	drain(t, clientConn, 1)

	if gotTopic != "chat.person.42" {
		t.Error(test.DiffMessage(gotTopic, "chat.person.42", "a Guard must see the concrete topic so it can authorize per-room/per-tenant"))
	}
	if gotPattern != "chat.person.*" {
		t.Error(test.DiffMessage(gotPattern, "chat.person.*", "a Guard must see which declared pattern matched"))
	}
	if gotConnID != "conn-guard" {
		t.Error(test.DiffMessage(gotConnID, "conn-guard", "a Guard must see the connection identity"))
	}
	if gotOperation != trace.OperationSubscribe {
		t.Error(test.DiffMessage(gotOperation, trace.OperationSubscribe, "a Guard must know which phase it is running in"))
	}
}

func TestPublish_GuardSeesTopicAndCanDenyPerTopic(t *testing.T) {
	ws := newTestWSBare(t)

	ws.eventMatcher.AddGuards("chat.*", common.BuildWSGuardMiddleware(func(c *ctx.WSContext) bool {
		return c.Topic() != "chat.forbidden"
	}))
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.allowed"}})
	drain(t, clientConn, 1)

	handlePublish(conn, ws, WSPayload{ID: "p1", Type: TypePublish, Topic: []string{"chat.allowed"}, Message: "ok"})
	if got := recvWSPayload(t, clientConn); got.Type != TypeEvent {
		t.Fatalf("expected the broker fan-out for the allowed topic, got %v", got.Type)
	}
	if got := recvWSPayload(t, clientConn); got.Type != TypeAck {
		t.Errorf("an allowed publish must be acked, got %v", got.Type)
	}

	// The guard denies chat.forbidden even though the pattern chat.* matches it.
	if err := ws.connmgr.Subscribe("conn-1", "chat.forbidden", func(*memorybroker.Message) {}); err != nil {
		t.Fatal(err)
	}
	handlePublish(conn, ws, WSPayload{ID: "p2", Type: TypePublish, Topic: []string{"chat.forbidden"}, Message: "nope"})
	if got := recvWSPayload(t, clientConn); got.Type != TypeError {
		t.Error(test.DiffMessage(got.Type, TypeError, "a per-topic Guard denial must reject the publish"))
	}
}

func TestUnsubscribe_RunsGuardAndCanBeDenied(t *testing.T) {
	ws := newTestWSBare(t)

	var guardRan bool
	ws.eventMatcher.AddGuards("locked.*", common.BuildWSGuardMiddleware(func(c *ctx.WSContext) bool {
		guardRan = true
		return c.Operation() != trace.OperationUnsubscribe
	}))
	ws.eventMatcher.AddInjectableHandler("locked.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"locked.room"}})
	drain(t, clientConn, 1)

	guardRan = false
	handleUnsubscribe(conn, ws, WSPayload{ID: "u1", Type: TypeUnsubscribe, Topic: []string{"locked.room"}})

	if !guardRan {
		t.Error(test.DiffMessage(false, true, "Unsubscribe must run the Guard chain, not bypass authorization"))
	}
	if got := recvWSPayload(t, clientConn); got.Type != TypeError {
		t.Error(test.DiffMessage(got.Type, TypeError, "a denied unsubscribe must be reported as an error"))
	}
	if !ws.connmgr.isSubscribed("conn-1", "locked.room") {
		t.Error(test.DiffMessage(false, true, "a denied unsubscribe must leave the subscription in place"))
	}
}

func TestUnsubscribe_AcksOnSuccess(t *testing.T) {
	ws := newTestWS(t, "chat.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.1"}})
	drain(t, clientConn, 1)

	handleUnsubscribe(conn, ws, WSPayload{ID: "u1", Type: TypeUnsubscribe, Topic: []string{"chat.1"}})

	got := recvWSPayload(t, clientConn)
	if got.Type != TypeAck {
		t.Fatal(test.DiffMessage(got.Type, TypeAck, "unsubscribe must confirm completion so the client is not left guessing"))
	}
	if got.ID != "u1" {
		t.Error(test.DiffMessage(got.ID, "u1", "the ack must echo the request id so the client can correlate it"))
	}
	if ws.connmgr.isSubscribed("conn-1", "chat.1") {
		t.Error(test.DiffMessage(true, false, "unsubscribe must actually remove the subscription"))
	}
}

func TestUnsubscribe_IsIdempotentAndScopedToTheOwningConnection(t *testing.T) {
	ws := newTestWS(t, "chat.*")

	serverA, clientA, cleanupA := newTestWSConnPair(t)
	defer cleanupA()
	serverB, clientB, cleanupB := newTestWSConnPair(t)
	defer cleanupB()

	connA := ws.connmgr.Register("conn-a", serverA)
	defer ws.connmgr.Unregister("conn-a")
	connB := ws.connmgr.Register("conn-b", serverB)
	defer ws.connmgr.Unregister("conn-b")

	handleSubscribe(connA, ws, WSPayload{ID: "sa", Type: TypeSubscribe, Topic: []string{"chat.shared"}})
	drain(t, clientA, 1)
	handleSubscribe(connB, ws, WSPayload{ID: "sb", Type: TypeSubscribe, Topic: []string{"chat.shared"}})
	drain(t, clientB, 1)

	handleUnsubscribe(connA, ws, WSPayload{ID: "ua", Type: TypeUnsubscribe, Topic: []string{"chat.shared"}})
	drain(t, clientA, 1)

	if !ws.connmgr.isSubscribed("conn-b", "chat.shared") {
		t.Error(test.DiffMessage(false, true, "one connection unsubscribing must not cancel another connection's subscription"))
	}

	handleUnsubscribe(connA, ws, WSPayload{ID: "ua2", Type: TypeUnsubscribe, Topic: []string{"chat.shared"}})
	if got := recvWSPayload(t, clientA); got.Type != TypeAck {
		t.Error(test.DiffMessage(got.Type, TypeAck, "unsubscribing twice must be idempotent, not an error"))
	}
}

func TestSubscribe_DoesNotRunInterceptors(t *testing.T) {
	ws := newTestWSBare(t)

	var interceptorCalls int
	ws.eventMatcher.AddInterceptors("chat.*", func(c *ctx.WSContext) {
		interceptorCalls++
		c.Next()
	})
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.1"}})
	drain(t, clientConn, 1)

	if interceptorCalls != 0 {
		t.Error(test.DiffMessage(interceptorCalls, 0, "Subscribe must run Guards only; an Interceptor's post phase can never run there, so running its pre phase leaves it half-executed"))
	}

	handleUnsubscribe(conn, ws, WSPayload{ID: "u1", Type: TypeUnsubscribe, Topic: []string{"chat.1"}})
	drain(t, clientConn, 1)

	if interceptorCalls != 0 {
		t.Error(test.DiffMessage(interceptorCalls, 0, "Unsubscribe must not run Interceptors either"))
	}
}

func TestPublish_RunsGuardThenInterceptorThenHandler(t *testing.T) {
	ws := newTestWSBare(t)

	var order []string
	ws.eventMatcher.AddGuards("chat.*", func(c *ctx.WSContext) {
		order = append(order, "guard")
		c.Next()
	})
	ws.eventMatcher.AddInterceptors("chat.*", func(c *ctx.WSContext) {
		order = append(order, "interceptor")
		c.Next()
	})
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {
		order = append(order, "handler")
	})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.1"}})
	drain(t, clientConn, 1)

	order = nil
	handlePublish(conn, ws, WSPayload{ID: "p1", Type: TypePublish, Topic: []string{"chat.1"}, Message: "x"})

	if len(order) != 3 || order[0] != "guard" || order[1] != "interceptor" || order[2] != "handler" {
		t.Error(test.DiffMessage(order, []string{"guard", "interceptor", "handler"}, "publish must run Guard -> Interceptor -> Handler in that order"))
	}
}

func TestSubscribe_MultiTopicReportsPerTopicOutcome(t *testing.T) {
	ws := newTestWS(t, "chat.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{
		ID:    "s1",
		Type:  TypeSubscribe,
		Topic: []string{"chat.ok", "nope.unknown", "chat.also-ok"},
	})

	got := recvWSPayload(t, clientConn)
	results := resultsOf(t, got)

	if len(results) != 3 {
		t.Fatal(test.DiffMessage(len(results), 3, "every requested topic must get its own outcome instead of the batch aborting at the first failure"))
	}
	if results[0]["status"] != trace.StatusOK || results[2]["status"] != trace.StatusOK {
		t.Error(test.DiffMessage(results, "ok, rejected, ok", "valid topics in a mixed batch must still be subscribed"))
	}
	if results[1]["status"] != trace.StatusRejected {
		t.Error(test.DiffMessage(results[1], "rejected", "the invalid topic must be reported as rejected"))
	}
	if !ws.connmgr.isSubscribed("conn-1", "chat.also-ok") {
		t.Error(test.DiffMessage(false, true, "a topic after the failing one must still be processed"))
	}
}

func TestSubscribe_DuplicateIsNoOp(t *testing.T) {
	ws := newTestWS(t, "chat.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	for i := 0; i < 3; i++ {
		handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.1"}})
		drain(t, clientConn, 1)
	}

	if n := ws.connmgr.SubscriptionCount("conn-1"); n != 1 {
		t.Error(test.DiffMessage(n, 1, "subscribing to the same topic repeatedly must not stack broker subscriptions"))
	}
}

func websocketJSONSend(conn *websocket.Conn, p WSPayload) error {
	return websocket.JSON.Send(conn, p)
}

func TestReadLoop_ClientPingIsAnsweredWithPongAndCorrelated(t *testing.T) {
	ws := newTestWSBare(t)
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	go readLoop(conn, ws)

	if err := websocketJSONSend(clientConn, WSPayload{ID: "hb-7", Type: TypePing}); err != nil {
		t.Fatal(err)
	}

	got := recvWSPayload(t, clientConn)
	if got.Type != TypePong {
		t.Fatal(test.DiffMessage(got.Type, TypePong, "the server must answer a client ping"))
	}
	if got.ID != "hb-7" {
		t.Error(test.DiffMessage(got.ID, "hb-7", "the pong must echo the ping id so the client can match it to its own heartbeat"))
	}
}

func registerAndSubscribe(t testing.TB, ws *WS, connID, topic string) (*WSConnection, *websocket.Conn, func()) {
	t.Helper()

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	conn := ws.connmgr.Register(connID, serverConn)
	if conn == nil {
		t.Fatalf("register %v failed", connID)
	}

	handleSubscribe(conn, ws, WSPayload{ID: "sub", Type: TypeSubscribe, Topic: []string{topic}})
	drain(t, clientConn, 1)

	return conn, clientConn, func() {
		ws.connmgr.Unregister(connID)
		cleanup()
	}
}

func TestAccessLog_SubscribeEmitsExactlyOneCompleteWithConnAndStatus(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {})

	_, _, cleanup := registerAndSubscribe(t, ws, "conn-obs", "chat.1")
	defer cleanup()

	completes := rec.byStage(trace.StageComplete)
	if len(completes) != 1 {
		t.Fatal(test.DiffMessage(len(completes), 1, "a subscribe must produce exactly one access-log entry"))
	}
	e := completes[0]
	if e.Operation != trace.OperationSubscribe || e.Target != "chat.1" {
		t.Error(test.DiffMessage(e.Operation+" "+e.Target, "subscribe chat.1", "the entry must name the phase and the topic"))
	}
	if e.Status != trace.StatusOK {
		t.Error(test.DiffMessage(e.Status, trace.StatusOK, "a successful subscribe must be marked ok"))
	}
	if e.ConnID != "conn-obs" {
		t.Error(test.DiffMessage(e.ConnID, "conn-obs", "without a connection id an access log cannot correlate operations on the same socket"))
	}
	if e.Transport != trace.TransportWS {
		t.Error(test.DiffMessage(e.Transport, trace.TransportWS, "WS operations must be tagged as WS transport"))
	}
}

func TestAccessLog_RejectedSubscribeIsStillLogged(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"forbidden.topic"}})
	drain(t, clientConn, 1)

	e := rec.completeFor(trace.OperationSubscribe, "forbidden.topic")
	if e == nil {
		t.Fatal(test.DiffMessage(nil, "complete event", "a rejected subscribe must appear in the access log, otherwise rejections are invisible in production"))
	}
	if e.Status != trace.StatusRejected {
		t.Error(test.DiffMessage(e.Status, trace.StatusRejected, "the entry must say it was rejected"))
	}
	if e.Code == 0 {
		t.Error(test.DiffMessage(e.Code, "non-zero", "the entry must carry the rejection code"))
	}
}

func TestAccessLog_UnsubscribeIsLogged(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.1"}})
	drain(t, clientConn, 1)
	handleUnsubscribe(conn, ws, WSPayload{ID: "u", Type: TypeUnsubscribe, Topic: []string{"chat.1"}})
	drain(t, clientConn, 1)

	e := rec.completeFor(trace.OperationUnsubscribe, "chat.1")
	if e == nil {
		t.Fatal(test.DiffMessage(nil, "complete event", "unsubscribe must be an access-logged phase like the other three"))
	}
	if e.Status != trace.StatusOK || e.ConnID != "conn-1" {
		t.Error(test.DiffMessage(e, "ok on conn-1", "the unsubscribe entry must carry status and connection id"))
	}
}

func TestAccessLog_PublishEmitsAHandlerStage(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func() ctx.Map { return ctx.Map{"ok": true} })

	conn, _, cleanup := registerAndSubscribe(t, ws, "conn-1", "chat.1")
	defer cleanup()

	handlePublish(conn, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.1"}, Message: "x"})

	handlerEvents := rec.byStage(trace.StageHandler)
	if len(handlerEvents) != 1 {
		t.Fatal(test.DiffMessage(len(handlerEvents), 1, "the WS handler must be timed like the HTTP one, otherwise the access log hides where the time went"))
	}
	if handlerEvents[0].Transport != trace.TransportWS {
		t.Error(test.DiffMessage(handlerEvents[0].Transport, trace.TransportWS, "the handler stage must be tagged WS"))
	}
	if handlerEvents[0].ConnID != "conn-1" {
		t.Error(test.DiffMessage(handlerEvents[0].ConnID, "conn-1", "the handler stage must carry the connection id"))
	}

	e := rec.completeFor(trace.OperationPublish, "chat.1")
	if e == nil || e.Status != trace.StatusOK {
		t.Error(test.DiffMessage(e, "publish ok", "a successful publish must be logged as ok"))
	}
}

func TestAccessLog_GuardDenialAndHandlerPanicAreDistinguishable(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddGuards("denied.*", common.BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return false }))
	ws.eventMatcher.AddInjectableHandler("denied.*", func() {})
	ws.eventMatcher.AddInjectableHandler("boom.*", func() { panic(exception.WSInternalErrorException("handler exploded")) })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"denied.room"}})
	drain(t, clientConn, 1)

	denied := rec.completeFor(trace.OperationSubscribe, "denied.room")
	if denied == nil || denied.Status != trace.StatusRejected {
		t.Error(test.DiffMessage(denied, "rejected", "a Guard denial must be logged as a rejection, not a failure"))
	}

	handleSubscribe(conn, ws, WSPayload{ID: "s2", Type: TypeSubscribe, Topic: []string{"boom.room"}})
	drain(t, clientConn, 1)
	handlePublish(conn, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"boom.room"}, Message: "x"})
	drain(t, clientConn, 1)

	blew := rec.completeFor(trace.OperationPublish, "boom.room")
	if blew == nil || blew.Status != trace.StatusFailed {
		t.Error(test.DiffMessage(blew, "failed", "a handler panic must be logged as a failure, distinct from an authorization rejection"))
	}
}

func TestAccessLog_ConnIDCorrelatesOperationsOnTheSameSocket(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func() {})

	conn, _, cleanup := registerAndSubscribe(t, ws, "conn-corr", "chat.1")
	defer cleanup()

	handlePublish(conn, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.1"}, Message: "x"})

	var ids []string
	for _, e := range rec.byStage(trace.StageComplete) {
		ids = append(ids, e.ConnID)
	}

	if len(ids) < 2 {
		t.Fatal(test.DiffMessage(len(ids), ">=2", "expected a subscribe and a publish entry"))
	}
	for _, id := range ids {
		if id != "conn-corr" {
			t.Fatal(test.DiffMessage(ids, "all conn-corr", "every operation on one socket must share a connection id so a production issue can be traced across the session"))
		}
	}
}

type wsAuditDTO struct {
	Room string `bind:"room"`
}

func (d wsAuditDTO) Transform(payload ctx.WSPayload, _ common.ArgumentMetadata) any {
	bound, _ := payload.Bind(d)
	dto := bound.(wsAuditDTO)

	dto.Room = strings.TrimSpace(dto.Room)
	if dto.Room == "" {
		panic(exception.InvalidPayloadException("room is required"))
	}

	return dto
}

func TestPipe_TransformsThePayloadBeforeTheHandlerRuns(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func(dto wsAuditDTO) ctx.Map {
		return ctx.Map{"room": dto.Room}
	})

	conn, clientConn, cleanup := registerAndSubscribe(t, ws, "conn-1", "chat.1")
	defer cleanup()

	handlePublish(conn, ws, WSPayload{
		ID:      "p",
		Type:    TypePublish,
		Topic:   []string{"chat.1"},
		Message: map[string]any{"room": "  lobby  "},
	})

	got := recvWSPayload(t, clientConn)
	if got.Type != TypeResponse {
		t.Fatalf("expected the handler response, got %v", got.Type)
	}
	m, ok := got.Message.(map[string]any)
	if !ok || m["room"] != "lobby" {
		t.Error(test.DiffMessage(got.Message, map[string]any{"room": "lobby"}, "the Pipe must run before the handler and hand it the transformed value"))
	}

	if len(rec.byStage(trace.StagePipe)) != 1 {
		t.Error(test.DiffMessage(len(rec.byStage(trace.StagePipe)), 1, "the pipe must be timed separately from the handler"))
	}
}

func TestPipe_PanicIsCaughtAndTheConnectionSurvives(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func(dto wsAuditDTO) ctx.Map {
		return ctx.Map{"room": dto.Room}
	})

	conn, clientConn, cleanup := registerAndSubscribe(t, ws, "conn-1", "chat.1")
	defer cleanup()

	handlePublish(conn, ws, WSPayload{
		ID:      "p1",
		Type:    TypePublish,
		Topic:   []string{"chat.1"},
		Message: map[string]any{"room": "   "},
	})

	got := recvWSPayload(t, clientConn)
	if got.Type != TypeError {
		t.Fatal(test.DiffMessage(got.Type, TypeError, "a Pipe that rejects an invalid payload must produce an error frame"))
	}

	if len(rec.byStage(trace.StagePipe)) != 1 {
		t.Error(test.DiffMessage(len(rec.byStage(trace.StagePipe)), 1, "a pipe that panics must still be traced, or its cost disappears from the access log"))
	}

	e := rec.completeFor(trace.OperationPublish, "chat.1")
	if e == nil || e.Status != trace.StatusFailed {
		t.Error(test.DiffMessage(e, "failed", "a rejected payload must be logged as a failed publish"))
	}

	// The connection must still be usable afterwards.
	handlePublish(conn, ws, WSPayload{
		ID:      "p2",
		Type:    TypePublish,
		Topic:   []string{"chat.1"},
		Message: map[string]any{"room": "lobby"},
	})
	if next := recvWSPayload(t, clientConn); next.Type != TypeResponse {
		t.Error(test.DiffMessage(next.Type, TypeResponse, "a Pipe failure must not tear down the connection"))
	}
}

func TestExceptionFilter_SeesTheTopicItIsHandling(t *testing.T) {
	rec := newTraceRecorder()
	ws := newTracedWS(t, rec)
	ws.eventMatcher.AddInjectableHandler("chat.*", func() { panic(exception.WSInternalErrorException("nope")) })

	var caughtTopic, caughtConnID string
	ws.catchFnsByEvent["chat.*"] = []common.WSCatch{
		func(c *ctx.WSContext, ex *exception.Exception) {
			caughtTopic = c.Topic()
			caughtConnID = c.ConnID()
			c.Send(ctx.Map{"code": ex.GetCode(), "topic": c.Topic()})
		},
	}

	conn, clientConn, cleanup := registerAndSubscribe(t, ws, "conn-1", "chat.99")
	defer cleanup()

	handlePublish(conn, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.99"}, Message: "x"})

	got := recvWSPayload(t, clientConn)
	if got.Type != TypeError {
		t.Fatalf("expected an error frame, got %v", got.Type)
	}
	if caughtTopic != "chat.99" {
		t.Error(test.DiffMessage(caughtTopic, "chat.99", "an exception filter must know which topic failed"))
	}
	if caughtConnID != "conn-1" {
		t.Error(test.DiffMessage(caughtConnID, "conn-1", "an exception filter must know which connection failed"))
	}
	if len(got.Topic) != 1 || got.Topic[0] != "chat.99" {
		t.Error(test.DiffMessage(got.Topic, []string{"chat.99"}, "the error frame must name the topic so the client can route it"))
	}
}

// subscribeAndDrainAck registers conn's subscription to topic via a real
// handleSubscribe call and drains the resulting ack frame from clientConn so
// later assertions only have to reason about event frames.
func subscribeAndDrainAck(t testing.TB, ws *WS, conn *WSConnection, clientConn *websocket.Conn, topic string) {
	t.Helper()

	handleSubscribe(conn, ws, WSPayload{ID: "sub-" + topic, Type: TypeSubscribe, Topic: []string{topic}})
	if p := recvWSPayload(t, clientConn); p.Type != TypeAck {
		t.Fatalf("expected subscribe ack for topic %q, got %v", topic, p.Type)
	}
}

// expectNoFrame asserts clientConn receives nothing within a short window,
// used to prove a subscriber that should NOT be fanned out to stays silent.
func expectNoFrame(t testing.TB, clientConn *websocket.Conn) {
	t.Helper()

	frame := make(chan WSPayload, 1)
	go func() {
		var p WSPayload
		if err := websocket.JSON.Receive(clientConn, &p); err == nil {
			frame <- p
		}
	}()

	select {
	case p := <-frame:
		t.Error(test.DiffMessage(p, nil, "expected no frame delivered, but one arrived"))
	case <-time.After(150 * time.Millisecond):
	}
}

// TestHandlePublish_FansOutToOverlappingWildcardSubscriber covers the fan-out
// requirement: publishing to a concrete topic (chat.456) must reach BOTH the
// exact subscriber of chat.456 AND the subscriber of the overlapping wildcard
// pattern chat.*, while a subscriber of an unrelated concrete topic
// (chat.123) must not receive anything.
func TestHandlePublish_FansOutToOverlappingWildcardSubscriber(t *testing.T) {
	ws := newTestWS(t, "chat.*")

	exactServer, exactClient, exactCleanup := newTestWSConnPair(t)
	defer exactCleanup()
	wildcardServer, wildcardClient, wildcardCleanup := newTestWSConnPair(t)
	defer wildcardCleanup()
	otherServer, otherClient, otherCleanup := newTestWSConnPair(t)
	defer otherCleanup()

	exactConn := ws.connmgr.Register("conn-exact", exactServer)
	defer ws.connmgr.Unregister("conn-exact")
	wildcardConn := ws.connmgr.Register("conn-wildcard", wildcardServer)
	defer ws.connmgr.Unregister("conn-wildcard")
	otherConn := ws.connmgr.Register("conn-other", otherServer)
	defer ws.connmgr.Unregister("conn-other")

	subscribeAndDrainAck(t, ws, exactConn, exactClient, "chat.456")
	subscribeAndDrainAck(t, ws, wildcardConn, wildcardClient, "chat.*")
	subscribeAndDrainAck(t, ws, otherConn, otherClient, "chat.123")

	handlePublish(exactConn, ws, WSPayload{ID: "pub-1", Type: TypePublish, Topic: []string{"chat.456"}, Message: "hello"})

	// exactConn is both publisher and a subscriber of chat.456, so it sees
	// the broker fan-out event before its own publish ack (see
	// TestHandlePublish_DeliversAfterSubscribe for why the ordering is
	// deterministic).
	eventFrame := recvWSPayload(t, exactClient)
	ackFrame := recvWSPayload(t, exactClient)
	if eventFrame.Type != TypeEvent || eventFrame.Message != "hello" {
		t.Error(test.DiffMessage(eventFrame, "event hello", "chat.456 subscriber should receive the published event"))
	}
	if ackFrame.Type != TypeAck {
		t.Error(test.DiffMessage(ackFrame.Type, TypeAck, "expected trailing publish ack"))
	}

	wildcardEvent := recvWSPayload(t, wildcardClient)
	if wildcardEvent.Type != TypeEvent || wildcardEvent.Message != "hello" || len(wildcardEvent.Topic) != 1 || wildcardEvent.Topic[0] != "chat.456" {
		t.Error(test.DiffMessage(wildcardEvent, "event chat.456 hello", "chat.* subscriber should be fanned out the chat.456 publish"))
	}

	expectNoFrame(t, otherClient)
}

// TestHandlePublish_LiteralWildcardTopicDoesNotExpand covers the other half
// of the fan-out requirement: Publish treats its topic as an exact string,
// never expanding it. Publishing the literal topic "chat.*" must reach only
// subscribers of that literal topic, never subscribers of concrete topics
// like chat.123 or chat.456 that happen to match the chat.* pattern.
func TestHandlePublish_LiteralWildcardTopicDoesNotExpand(t *testing.T) {
	ws := newTestWS(t, "chat.*")

	wildcardServer, wildcardClient, wildcardCleanup := newTestWSConnPair(t)
	defer wildcardCleanup()
	otherServer, otherClient, otherCleanup := newTestWSConnPair(t)
	defer otherCleanup()

	wildcardConn := ws.connmgr.Register("conn-wildcard", wildcardServer)
	defer ws.connmgr.Unregister("conn-wildcard")
	otherConn := ws.connmgr.Register("conn-other", otherServer)
	defer ws.connmgr.Unregister("conn-other")

	subscribeAndDrainAck(t, ws, wildcardConn, wildcardClient, "chat.*")
	subscribeAndDrainAck(t, ws, otherConn, otherClient, "chat.123")

	handlePublish(wildcardConn, ws, WSPayload{ID: "pub-1", Type: TypePublish, Topic: []string{"chat.*"}, Message: "literal"})

	eventFrame := recvWSPayload(t, wildcardClient)
	ackFrame := recvWSPayload(t, wildcardClient)
	if eventFrame.Type != TypeEvent || eventFrame.Message != "literal" {
		t.Error(test.DiffMessage(eventFrame, "event literal", "chat.* subscriber should receive a publish to the literal chat.* topic"))
	}
	if ackFrame.Type != TypeAck {
		t.Error(test.DiffMessage(ackFrame.Type, TypeAck, "expected trailing publish ack"))
	}

	expectNoFrame(t, otherClient)
}

// TestFanout_ProductionScenario is the scenario from the WebSocket Controller
// spec: A and B both listen on chat.123, C listens on chat.456, then A
// publishes to chat.123. A must get its handler's answer, B must get the
// fan-out, C must stay silent.
func TestFanout_ProductionScenario(t *testing.T) {
	ws := newTestWSBare(t)
	ws.eventMatcher.AddInjectableHandler("chat.*", func(topic ctx.WSTopic) ctx.Map {
		return ctx.Map{"handled": string(topic)}
	})

	serverA, clientA, cleanupA := newTestWSConnPair(t)
	defer cleanupA()
	serverB, clientB, cleanupB := newTestWSConnPair(t)
	defer cleanupB()
	serverC, clientC, cleanupC := newTestWSConnPair(t)
	defer cleanupC()

	connA := ws.connmgr.Register("conn-a", serverA)
	defer ws.connmgr.Unregister("conn-a")
	connB := ws.connmgr.Register("conn-b", serverB)
	defer ws.connmgr.Unregister("conn-b")
	connC := ws.connmgr.Register("conn-c", serverC)
	defer ws.connmgr.Unregister("conn-c")

	handleSubscribe(connA, ws, WSPayload{ID: "sa", Type: TypeSubscribe, Topic: []string{"chat.123"}})
	drain(t, clientA, 1)
	handleSubscribe(connB, ws, WSPayload{ID: "sb", Type: TypeSubscribe, Topic: []string{"chat.123"}})
	drain(t, clientB, 1)
	handleSubscribe(connC, ws, WSPayload{ID: "sc", Type: TypeSubscribe, Topic: []string{"chat.456"}})
	drain(t, clientC, 1)

	handlePublish(connA, ws, WSPayload{ID: "p1", Type: TypePublish, Topic: []string{"chat.123"}, Message: "hello"})

	response := recvWSPayload(t, clientA)
	if response.Type != TypeResponse {
		t.Fatal(test.DiffMessage(response.Type, TypeResponse, "the publisher must receive its handler's return value as a response frame, distinguishable from a fan-out event"))
	}
	if len(response.Topic) != 1 || response.Topic[0] != "chat.123" {
		t.Error(test.DiffMessage(response.Topic, []string{"chat.123"}, "a handler response must name the topic it answers so a client can route it"))
	}
	if m, ok := response.Message.(map[string]any); !ok || m["handled"] != "chat.123" {
		t.Error(test.DiffMessage(response.Message, map[string]any{"handled": "chat.123"}, "the handler must be able to see the concrete topic it is serving"))
	}

	fanOutA := recvWSPayload(t, clientA)
	if fanOutA.Type != TypeEvent || fanOutA.Message != "hello" {
		t.Errorf("publisher should also observe its own fan-out event, got %+v", fanOutA)
	}
	if ack := recvWSPayload(t, clientA); ack.Type != TypeAck {
		t.Errorf("publisher must be acked, got %v", ack.Type)
	}

	fanOutB := recvWSPayload(t, clientB)
	if fanOutB.Type != TypeEvent || fanOutB.Message != "hello" {
		t.Error(test.DiffMessage(fanOutB, "event hello", "the other subscriber of chat.123 must receive the published message"))
	}
	if fanOutB.Pattern != "chat.123" {
		t.Error(test.DiffMessage(fanOutB.Pattern, "chat.123", "a fan-out event must name the subscription it was delivered for"))
	}

	expectNoFrame(t, clientC)
}

func TestFanout_WildcardSubscriberGetsPatternForClientSideRouting(t *testing.T) {
	ws := newTestWS(t, "chat.*")

	serverPub, clientPub, cleanupPub := newTestWSConnPair(t)
	defer cleanupPub()
	serverSub, clientSub, cleanupSub := newTestWSConnPair(t)
	defer cleanupSub()

	connPub := ws.connmgr.Register("conn-pub", serverPub)
	defer ws.connmgr.Unregister("conn-pub")
	connSub := ws.connmgr.Register("conn-sub", serverSub)
	defer ws.connmgr.Unregister("conn-sub")

	handleSubscribe(connSub, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.*"}})
	drain(t, clientSub, 1)
	handleSubscribe(connPub, ws, WSPayload{ID: "sp", Type: TypeSubscribe, Topic: []string{"chat.777"}})
	drain(t, clientPub, 1)

	handlePublish(connPub, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.777"}, Message: "wild"})

	got := recvWSPayload(t, clientSub)
	if len(got.Topic) != 1 || got.Topic[0] != "chat.777" {
		t.Error(test.DiffMessage(got.Topic, []string{"chat.777"}, "the event must carry the concrete topic"))
	}
	if got.Pattern != "chat.*" {
		t.Error(test.DiffMessage(got.Pattern, "chat.*", "a wildcard subscriber cannot route the event without the pattern it subscribed with"))
	}
}

func TestPublish_DoesNotDuplicateDeliveryOnRepeatSubscribe(t *testing.T) {
	ws := newTestWS(t, "chat.*")

	serverA, clientA, cleanupA := newTestWSConnPair(t)
	defer cleanupA()
	serverB, clientB, cleanupB := newTestWSConnPair(t)
	defer cleanupB()

	connA := ws.connmgr.Register("conn-a", serverA)
	defer ws.connmgr.Unregister("conn-a")
	connB := ws.connmgr.Register("conn-b", serverB)
	defer ws.connmgr.Unregister("conn-b")

	for i := 0; i < 3; i++ {
		handleSubscribe(connB, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.1"}})
		drain(t, clientB, 1)
	}
	handleSubscribe(connA, ws, WSPayload{ID: "sa", Type: TypeSubscribe, Topic: []string{"chat.1"}})
	drain(t, clientA, 1)

	handlePublish(connA, ws, WSPayload{ID: "p1", Type: TypePublish, Topic: []string{"chat.1"}, Message: "once"})

	if got := recvWSPayload(t, clientB); got.Type != TypeEvent || got.Message != "once" {
		t.Fatalf("expected one fan-out event, got %+v", got)
	}
	expectNoFrame(t, clientB)
}

func TestSubscribe_EnforcesPerConnectionSubscriptionLimit(t *testing.T) {
	ws := newTestWS(t, "chat.*")
	ws.connmgr.maxSubsPerConn = 3

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	for i := 0; i < 3; i++ {
		handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat." + string(rune('a'+i))}})
		drain(t, clientConn, 1)
	}

	handleSubscribe(conn, ws, WSPayload{ID: "s-over", Type: TypeSubscribe, Topic: []string{"chat.overflow"}})
	got := recvWSPayload(t, clientConn)

	if got.Type != TypeError {
		t.Fatal(test.DiffMessage(got.Type, TypeError, "an unbounded subscription count is a denial-of-service vector; the limit must be enforced"))
	}
	if n := ws.connmgr.SubscriptionCount("conn-1"); n != 3 {
		t.Error(test.DiffMessage(n, 3, "a rejected subscribe must not be recorded"))
	}
}

func TestSubscribe_RejectsEmptyAndOversizedTopics(t *testing.T) {
	ws := newTestWS(t, "*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	oversized := make([]byte, maxWSTopicLength+1)
	for i := range oversized {
		oversized[i] = 'a'
	}

	for _, topic := range []string{"", string(oversized)} {
		handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{topic}})
		if got := recvWSPayload(t, clientConn); got.Type != TypeError {
			t.Errorf("topic of length %d must be rejected, got %v", len(topic), got.Type)
		}
	}
}

func drainForever(conn *websocket.Conn) {
	for {
		var p WSPayload
		if err := websocket.JSON.Receive(conn, &p); err != nil {
			return
		}
	}
}

func TestConcurrency_SubscribePublishUnsubscribeCloseUnderLoad(t *testing.T) {
	ws := newTestWS(t, "chat.*")

	const conns = 8
	const rounds = 40

	var wg sync.WaitGroup
	for i := 0; i < conns; i++ {
		serverConn, clientConn, cleanup := newTestWSConnPair(t)
		defer cleanup()

		connID := "conn-" + string(rune('a'+i))
		conn := ws.connmgr.Register(connID, serverConn)
		if conn == nil {
			t.Fatalf("register %v failed", connID)
		}

		go drainForever(clientConn)

		wg.Add(1)
		go func(conn *WSConnection, id string) {
			defer wg.Done()
			for r := 0; r < rounds; r++ {
				handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.shared", "chat." + id}})
				handlePublish(conn, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.shared"}, Message: r})
				handleUnsubscribe(conn, ws, WSPayload{ID: "u", Type: TypeUnsubscribe, Topic: []string{"chat." + id}})
				ws.connmgr.touch(id)
				ws.connmgr.Stats()
			}
			ws.connmgr.Unregister(id)
		}(conn, connID)
	}

	wg.Wait()

	if stats := ws.connmgr.Stats(); stats.Connections != 0 || stats.Subscriptions != 0 {
		t.Error(test.DiffMessage(stats, WSStats{}, "after every connection closed, no connection or subscription state may remain"))
	}
}

func TestConcurrency_PublishRacingWithEviction(t *testing.T) {
	ws := newTestWS(t, "chat.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	go drainForever(clientConn)

	conn := ws.connmgr.Register("conn-1", serverConn)
	handleSubscribeDirect(t, ws, "conn-1", "chat.1")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 200; i++ {
			handlePublish(conn, ws, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.1"}, Message: i})
		}
	}()
	go func() {
		defer wg.Done()
		time.Sleep(2 * time.Millisecond)
		ws.connmgr.Evict("conn-1")
	}()

	wg.Wait()

	if _, alive := ws.connmgr.Get("conn-1"); alive {
		t.Error(test.DiffMessage(true, false, "the evicted connection must stay gone even while a publish was in flight"))
	}
}
