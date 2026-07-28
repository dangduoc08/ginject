package core

import (
	"reflect"
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
		data, _ := invokeWSHandlerByProviders(f, nil, c, ev)
		return data
	}
	ws.emitPostInterceptor = func(c *ctx.WSContext, name string, duration time.Duration) {}
	ws.emitComplete = func(c *ctx.WSContext, operation, target string) {}

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
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})

	var got WSPayload
	if err := websocket.JSON.Receive(clientConn, &got); err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got.Type != TypeAck {
		t.Error(test.DiffMessage(got.Type, TypeAck, "subscribing to a topic matching a registered pattern should ack"))
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
	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

	// Expect 3 frames on the wire: the subscribe ack, the publish ack, and
	// the event the broker fans back out to this same connection (it's
	// subscribed to the topic it just published to).
	results := make(chan WSPayload, 3)
	go func() {
		for i := 0; i < 3; i++ {
			var p WSPayload
			if err := websocket.JSON.Receive(clientConn, &p); err != nil {
				return
			}
			results <- p
		}
	}()

	var acks, events int
	deadline := time.After(2 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case p := <-results:
			switch p.Type {
			case TypeAck:
				acks++
			case TypeEvent:
				events++
				if len(p.Topic) != 1 || p.Topic[0] != "chat.to.user2" || p.Message != "hi" {
					t.Error(test.DiffMessage(p, "event chat.to.user2 hi", "unexpected event payload delivered via broker"))
				}
			}
		case <-deadline:
			t.Fatalf("timed out waiting for frames, acks=%d events=%d", acks, events)
		}
	}

	if acks != 2 {
		t.Error(test.DiffMessage(acks, 2, "expected 2 acks (subscribe + publish)"))
	}
	if events != 1 {
		t.Error(test.DiffMessage(events, 1, "expected exactly 1 event delivered via broker"))
	}
}

func TestDispatchWSEvent_HandlerReturnValueRepliesAsTypeEvent(t *testing.T) {
	ws := newTestWSBare(t)
	ws.eventMatcher.AddInjectableHandler("chat.to.*", func() ctx.Map {
		return ctx.Map{"reply": "ack-from-handler"}
	})

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})
	if p := recvWSPayload(t, clientConn); p.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", p.Type)
	}

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})

	// Broker.Publish runs synchronously and TrySend is a single FIFO
	// channel drained by one writer goroutine, so frame order on the wire
	// is deterministic: dispatchWSEvent's own TypeEvent reply (the
	// handler's return value) first, then the broker fan-out of the
	// original published message, then the trailing publish ack.
	handlerReplyFrame := recvWSPayload(t, clientConn)
	fanOutFrame := recvWSPayload(t, clientConn)
	ackFrame := recvWSPayload(t, clientConn)

	if handlerReplyFrame.Type != TypeEvent {
		t.Fatalf("expected first frame to be the handler's TypeEvent reply, got %v", handlerReplyFrame.Type)
	}
	handlerReply, ok := handlerReplyFrame.Message.(map[string]any)
	if !ok || handlerReply["reply"] != "ack-from-handler" {
		t.Error(test.DiffMessage(handlerReplyFrame.Message, map[string]any{"reply": "ack-from-handler"}, "handler's return value should be sent back via TypeEvent"))
	}

	if fanOutFrame.Type != TypeEvent || fanOutFrame.Message != "hi" {
		t.Error(test.DiffMessage(fanOutFrame, "TypeEvent hi", "broker fan-out should still deliver the original published message"))
	}

	if ackFrame.Type != TypeAck {
		t.Error(test.DiffMessage(ackFrame.Type, TypeAck, "expected trailing publish ack"))
	}
}

// Guard/Interceptor middlewares run on both subscribe and publish (only the
// global handshake-time middlewares are subscribe/publish-exempt), so a
// Guard that unconditionally denies blocks subscribe itself — there's no
// way to reach publish's must-already-be-subscribed check at all.
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
	if p := recvWSPayload(t, clientConn); p.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", p.Type)
	}

	// Registered only now, after subscribe succeeded, so this Guard denies
	// publish specifically without blocking the subscribe step above.
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
	if p := recvWSPayload(t, clientConn); p.Type != TypeAck {
		t.Fatalf("expected subscribe ack, got %v", p.Type)
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
	got := recvWSPayload(t, clientConn)
	if got.Type != TypeAck {
		t.Error(test.DiffMessage(got.Type, TypeAck, "readLoop should dispatch a subscribe message to handleSubscribe"))
	}

	if err := websocket.JSON.Send(clientConn, WSPayload{ID: "req-2", Type: "bogus"}); err != nil {
		t.Fatal(err)
	}
	got = recvWSPayload(t, clientConn)
	if got.Type != TypeError {
		t.Error(test.DiffMessage(got.Type, TypeError, "readLoop should reply with an error for an unsupported payload type"))
	}

	_ = clientConn.Close()
	<-done
}

func TestHandleUnsubscribe_RemovesTopicAndAcks(t *testing.T) {
	ws := newTestWS(t, "chat.to.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})
	_ = recvWSPayload(t, clientConn) // ack from subscribe

	handleUnsubscribe(conn, ws.connmgr, WSPayload{ID: "req-2", Topic: []string{"chat.to.user2"}})
	got := recvWSPayload(t, clientConn)

	if got.Type != TypeAck {
		t.Error(test.DiffMessage(got.Type, TypeAck, "handleUnsubscribe should ack once all topics are removed"))
	}
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

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})
	_ = recvWSPayload(t, clientConn)

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
	_ = recvWSPayload(t, clientConn)

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
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	pre = nil
	post = nil
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
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
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	preCount = 0
	postCount = 0
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
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
	ws.emitComplete = func(c *ctx.WSContext, operation, target string) {
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
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	got = nil
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
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
	ws.emitComplete = func(c *ctx.WSContext, operation, target string) {
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
	_ = recvWSPayload(t, clientConn)

	mu.Lock()
	guardID, completeID = "", ""
	mu.Unlock()

	handlePublish(conn, ws, WSPayload{ID: "req-2", Type: TypePublish, Topic: []string{"chat.to.user2"}, Message: "hi"})
	_ = recvWSPayload(t, clientConn)
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
	ws.emitComplete = func(c *ctx.WSContext, operation, target string) {
		ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StageComplete, Transport: trace.TransportWS, Operation: operation, Target: target})
	}

	pattern := "chat.to.*"
	ws.eventMatcher.AddInjectableHandler(pattern, func() string { return "ok" })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})
	_ = recvWSPayload(t, clientConn)

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
	ws.emitComplete = func(c *ctx.WSContext, operation, target string) {
		ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StageComplete, Transport: trace.TransportWS, Operation: operation, Target: target})
	}

	pattern := "chat.to.*"
	ws.eventMatcher.AddInjectableHandler(pattern, func() { panic("boom") })

	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "req-1", Type: TypeSubscribe, Topic: []string{"chat.to.user2"}})
	_ = recvWSPayload(t, clientConn)

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
	ws.emitComplete = func(c *ctx.WSContext, operation, target string) {
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
	_ = recvWSPayload(t, clientConn)

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
