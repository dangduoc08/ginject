package core

import (
	"fmt"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/memorybroker"
)

// newTestWSConnPair spins up a real HTTP server upgraded to WebSocket and
// dials it, returning the server-side and client-side *websocket.Conn. The
// server-side handler blocks until cleanup() closes it, so the connection
// stays alive for the duration of the test.
func newTestWSConnPair(t testing.TB) (server *websocket.Conn, client *websocket.Conn, cleanup func()) {
	t.Helper()

	serverConnCh := make(chan *websocket.Conn, 1)
	handlerDone := make(chan struct{})

	httpServer := httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		serverConnCh <- c
		<-handlerDone
	}))

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	clientConn, err := websocket.Dial(wsURL, "", httpServer.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	serverConn := <-serverConnCh

	return serverConn, clientConn, func() {
		close(handlerDone)
		_ = clientConn.Close()
		httpServer.Close()
	}
}

func TestWSConnection_TrySend_DeliversToClient(t *testing.T) {
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	conn := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	ok := conn.TrySend(WSPayload{Type: TypeEvent, Topic: []string{"chat.to.user2"}, Message: "hi"})
	if !ok {
		t.Error(test.DiffMessage(ok, true, "TrySend should succeed with room in the buffer"))
	}

	var got WSPayload
	if err := websocket.JSON.Receive(clientConn, &got); err != nil {
		t.Fatalf("client receive: %v", err)
	}

	if got.Type != TypeEvent || len(got.Topic) != 1 || got.Topic[0] != "chat.to.user2" {
		t.Error(test.DiffMessage(got, "event/chat.to.user2", "unexpected payload delivered to client"))
	}
}

func TestWSConnection_TrySend_ConcurrentSendsNoRace(t *testing.T) {
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	conn := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	const goroutines = 16
	const perGoroutine = 8
	total := goroutines * perGoroutine

	// TrySend is non-blocking and may legitimately drop payloads under a
	// burst this size (DefaultWSSendBufferSize is 32) — that's the contract, not a
	// bug. What must hold under -race is: whatever TrySend *did* accept
	// (returned true) is exactly what the client receives, with no panic
	// and no data race, regardless of how many goroutines call it at once.
	received := make(chan struct{}, total)
	stopReceiving := make(chan struct{})
	go func() {
		for {
			var got WSPayload
			if err := websocket.JSON.Receive(clientConn, &got); err != nil {
				return
			}
			select {
			case received <- struct{}{}:
			case <-stopReceiving:
				return
			}
		}
	}()

	var accepted int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				if conn.TrySend(WSPayload{Type: TypeEvent, Message: "x"}) {
					atomic.AddInt64(&accepted, 1)
				}
			}
		}()
	}
	wg.Wait()

	want := int(atomic.LoadInt64(&accepted))
	if want == 0 {
		t.Fatal("TrySend accepted 0 payloads out of 128 concurrent attempts — buffer size regression?")
	}

	deadline := time.After(2 * time.Second)
	count := 0
	for count < want {
		select {
		case <-received:
			count++
		case <-deadline:
			t.Fatalf("timed out waiting for deliveries, got %d/%d accepted", count, want)
		}
	}
	close(stopReceiving)
}

func TestWSConnection_TrySend_DropsWhenBufferFull(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	// No client-side reads happen in this test, so once the writer goroutine
	// blocks on its own in-flight websocket.JSON.Send, the buffer fills up
	// and TrySend must start returning false instead of blocking forever.
	conn := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	done := make(chan bool)
	go func() {
		sawDrop := false
		for i := 0; i < DefaultWSSendBufferSize*4; i++ {
			if !conn.TrySend(WSPayload{Type: TypeEvent, Message: "x"}) {
				sawDrop = true
			}
		}
		done <- sawDrop
	}()

	select {
	case sawDrop := <-done:
		if !sawDrop {
			t.Error(test.DiffMessage(sawDrop, true, "TrySend should drop at least one payload once the buffer is full"))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("TrySend blocked instead of dropping — it must never block the caller")
	}
}

func TestWSConnmgr_UnregisterStopsWriterWithoutPanic(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	conn := connmgr.Register("conn-1", serverConn)

	connmgr.Unregister("conn-1")

	// A send racing with (or arriving after) Unregister must not panic —
	// done is closed, not send, precisely so this stays safe.
	conn.TrySend(WSPayload{Type: TypeEvent, Message: "after-unregister"})
}

func TestWSConnmgr_Get(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	registered := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	got, ok := connmgr.Get("conn-1")
	if !ok || got != registered {
		t.Error(test.DiffMessage(got, registered, "Get should return the connection registered under that id"))
	}

	if _, ok := connmgr.Get("missing"); ok {
		t.Error(test.DiffMessage(ok, false, "Get should report false for an unregistered id"))
	}
}

func TestWSConnmgr_Touch(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	conn := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	before := conn.LastSeenAt()
	time.Sleep(time.Millisecond)
	connmgr.touch("conn-1")

	got, _ := connmgr.Get("conn-1")
	if !got.LastSeenAt().After(before) {
		t.Error(test.DiffMessage(got.LastSeenAt(), "after "+before.String(), "touch should advance LastSeen"))
	}
}

func TestWSConnmgr_TouchUnknownConnIsNoop(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	connmgr.touch("missing")
}

func TestWSConnmgr_UnsubscribeRemovesOnlyMatchingTopic(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)

	if err := connmgr.Subscribe("conn-1", "topic.a", func(*memorybroker.Message) {}); err != nil {
		t.Fatal(err)
	}
	if err := connmgr.Subscribe("conn-1", "topic.b", func(*memorybroker.Message) {}); err != nil {
		t.Fatal(err)
	}

	if err := connmgr.Unsubscribe("conn-1", "topic.a"); err != nil {
		t.Fatal(err)
	}

	if connmgr.isSubscribed("conn-1", "topic.a") {
		t.Error(test.DiffMessage(true, false, "Unsubscribe should remove the given topic"))
	}
	if !connmgr.isSubscribed("conn-1", "topic.b") {
		t.Error(test.DiffMessage(false, true, "Unsubscribe should leave other topics untouched"))
	}
}

func TestWSConnmgr_UnsubscribeUnknownTopicIsNoop(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	if err := connmgr.Unsubscribe("conn-1", "never-subscribed"); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe on a topic never subscribed to should not error"))
	}
}

type countingLogger struct {
	warns  atomic.Int64
	errors atomic.Int64
}

func (l *countingLogger) Info(string, ...any) {}

func (l *countingLogger) Debug(string, ...any) {}

func (l *countingLogger) Fatal(string, ...any) {}

func (l *countingLogger) Warn(string, ...any) { l.warns.Add(1) }

func (l *countingLogger) Error(string, ...any) { l.errors.Add(1) }

func goroutineCount() int {
	return runtime.NumGoroutine()
}

func handleSubscribeDirect(t testing.TB, ws *WS, connID, topic string) {
	t.Helper()
	if err := ws.connmgr.Subscribe(connID, topic, func(*memorybroker.Message) {}); err != nil {
		t.Fatal(err)
	}
}

func TestClose_CleansUpEverySubscription(t *testing.T) {
	ws := newTestWS(t, "chat.*")
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.1", "chat.2", "chat.3"}})
	drain(t, clientConn, 1)

	if n := ws.connmgr.SubscriptionCount("conn-1"); n != 3 {
		t.Fatal(test.DiffMessage(n, 3, "all three topics should be subscribed"))
	}

	ws.connmgr.Unregister("conn-1")

	if n := ws.connmgr.SubscriptionCount("conn-1"); n != 0 {
		t.Error(test.DiffMessage(n, 0, "closing a connection must release every one of its broker subscriptions"))
	}
	if stats := ws.connmgr.Stats(); stats.Subscriptions != 0 || stats.Connections != 0 {
		t.Error(test.DiffMessage(stats, WSStats{}, "a closed connection must leave no residue in the manager"))
	}
}

func TestBackpressure_DroppedFramesAreCountedAndSurfaced(t *testing.T) {
	ws := NewWS(&WSConfig{logger: &countingLogger{}, SendBufferSize: 2, MaxDroppedFrames: 0})
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-slow", serverConn)
	defer ws.connmgr.Unregister("conn-slow")

	// The writer drains at most one frame before blocking on the unread peer,
	// so filling well past the buffer guarantees drops.
	for i := 0; i < 64; i++ {
		ws.connmgr.trySend(conn, WSPayload{Type: TypeEvent, Message: i})
	}

	if conn.Dropped() == 0 {
		t.Fatal(test.DiffMessage(0, ">0", "frames dropped for a slow consumer must be counted, not lost silently"))
	}
	if stats := ws.connmgr.Stats(); stats.SlowConsumers != 1 {
		t.Error(test.DiffMessage(stats.SlowConsumers, 1, "a connection that overflowed its queue must be observable as a slow consumer"))
	}
}

func TestBackpressure_SlowConsumerIsEvictedPastTheLimit(t *testing.T) {
	logger := &countingLogger{}
	ws := NewWS(&WSConfig{logger: logger, SendBufferSize: 2, MaxDroppedFrames: 8})
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-slow", serverConn)
	handleSubscribeDirect(t, ws, "conn-slow", "chat.1")

	for i := 0; i < 200; i++ {
		ws.connmgr.trySend(conn, WSPayload{Type: TypeEvent, Message: i})
		if _, alive := ws.connmgr.Get("conn-slow"); !alive {
			break
		}
	}

	if _, alive := ws.connmgr.Get("conn-slow"); alive {
		t.Fatal(test.DiffMessage(true, false, "a consumer that keeps dropping frames must be evicted rather than losing messages forever"))
	}
	if !conn.Closed() {
		t.Error(test.DiffMessage(false, true, "an evicted connection must be marked closed"))
	}
	if n := ws.connmgr.SubscriptionCount("conn-slow"); n != 0 {
		t.Error(test.DiffMessage(n, 0, "eviction must release the connection's subscriptions"))
	}
	if logger.errors.Load() == 0 {
		t.Error(test.DiffMessage(0, ">0", "evicting a slow consumer must be logged so it is diagnosable in production"))
	}
}

func TestHeartbeat_StaleConnectionIsReapedAndCleanedUp(t *testing.T) {
	logger := &countingLogger{}
	ws := NewWS(&WSConfig{logger: logger})
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-stale", serverConn)
	handleSubscribeDirect(t, ws, "conn-stale", "chat.1")

	reaped := ws.connmgr.reapDeadConns(time.Hour)
	if len(reaped) != 0 {
		t.Fatal(test.DiffMessage(reaped, []string{}, "a fresh connection must not be reaped"))
	}

	conn.lastSeen.Store(time.Now().Add(-2 * time.Minute).UnixNano())
	reaped = ws.connmgr.reapDeadConns(time.Minute)

	if len(reaped) != 1 || reaped[0] != "conn-stale" {
		t.Fatal(test.DiffMessage(reaped, []string{"conn-stale"}, "a connection that stopped answering must be reaped"))
	}
	if _, alive := ws.connmgr.Get("conn-stale"); alive {
		t.Error(test.DiffMessage(true, false, "a reaped connection must be removed from the manager"))
	}
	if n := ws.connmgr.SubscriptionCount("conn-stale"); n != 0 {
		t.Error(test.DiffMessage(n, 0, "reaping must release the connection's subscriptions"))
	}
	if logger.warns.Load() == 0 {
		t.Error(test.DiffMessage(0, ">0", "a heartbeat timeout must be logged"))
	}
}

func TestHeartbeat_InboundTrafficKeepsTheConnectionAlive(t *testing.T) {
	ws := NewWS(&WSConfig{logger: &countingLogger{}})
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	defer ws.connmgr.Unregister("conn-1")

	conn.lastSeen.Store(time.Now().Add(-2 * time.Minute).UnixNano())
	ws.connmgr.touch("conn-1")

	if reaped := ws.connmgr.reapDeadConns(time.Minute); len(reaped) != 0 {
		t.Error(test.DiffMessage(reaped, []string{}, "a connection that just sent traffic must not be reaped"))
	}
}

func TestHeartbeat_ReaperGoroutineIsNotStartedWithoutAShutdownChannel(t *testing.T) {
	connmgr := NewWSConnmgr(&countingLogger{}, nil)
	before := goroutineCount()
	connmgr.startDeadConnDetection(time.Millisecond, time.Millisecond, nil)
	time.Sleep(20 * time.Millisecond)

	if after := goroutineCount(); after > before {
		t.Error(test.DiffMessage(after, before, "a reaper with no shutdown channel could never be stopped, so it must not be started"))
	}
}

func TestConcurrency_SubscribeAfterEvictionDoesNotResurrectState(t *testing.T) {
	ws := newTestWS(t, "chat.*")
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	conn := ws.connmgr.Register("conn-1", serverConn)
	ws.connmgr.Evict("conn-1")

	handleSubscribe(conn, ws, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.1"}})

	if n := ws.connmgr.SubscriptionCount("conn-1"); n != 0 {
		t.Error(test.DiffMessage(n, 0, "a subscribe that lands after the connection died must not leak a broker subscription"))
	}
}

func wslTestConn(t *testing.T) (*websocket.Conn, func()) {
	t.Helper()

	srvConnCh := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(websocket.Handler(func(c *websocket.Conn) {
		srvConnCh <- c
		<-make(chan struct{})
	}))

	clientConn, err := websocket.Dial("ws"+srv.URL[len("http"):], "", srv.URL)
	if err != nil {
		srv.Close()
		t.Fatal(err)
	}

	select {
	case sc := <-srvConnCh:
		return sc, func() {
			_ = clientConn.Close()
			srv.Close()
		}
	case <-time.After(3 * time.Second):
		srv.Close()
		t.Fatal("server side of websocket never arrived")
		return nil, func() {}
	}
}

func TestWSConnmgr_Register_RejectsBeyondMaxConnections(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	connmgr.maxConns = 2

	c1, done1 := wslTestConn(t)
	defer done1()
	c2, done2 := wslTestConn(t)
	defer done2()
	c3, done3 := wslTestConn(t)
	defer done3()

	if got := connmgr.Register("a", c1); got == nil {
		t.Fatal(test.DiffMessage(nil, "connection", "first connection must be accepted"))
	}
	if got := connmgr.Register("b", c2); got == nil {
		t.Fatal(test.DiffMessage(nil, "connection", "second connection must be accepted"))
	}
	if got := connmgr.Register("c", c3); got != nil {
		t.Error(test.DiffMessage("connection", nil, "a connection beyond maxConns must be rejected"))
	}
	if connmgr.Count() != 2 {
		t.Error(test.DiffMessage(connmgr.Count(), 2, "a rejected connection must not be tracked"))
	}

	connmgr.Unregister("a")
	if got := connmgr.Register("c", c3); got == nil {
		t.Error(test.DiffMessage(nil, "connection", "a slot freed by Unregister must be reusable"))
	}
	connmgr.Unregister("b")
	connmgr.Unregister("c")
}

func TestWSConnmgr_Register_RefusesDuplicateID(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)

	c1, done1 := wslTestConn(t)
	defer done1()
	c2, done2 := wslTestConn(t)
	defer done2()

	first := connmgr.Register("same", c1)
	if first == nil {
		t.Fatal(test.DiffMessage(nil, "connection", "first connection must be accepted"))
	}

	if got := connmgr.Register("same", c2); got != nil {
		t.Error(test.DiffMessage("connection", nil, "a duplicate connID must be refused, never evict the live connection"))
	}

	if existing, ok := connmgr.Get("same"); !ok || existing != first {
		t.Error(test.DiffMessage(existing, first, "the originally registered connection must stay in place"))
	}

	connmgr.Unregister("same")
}

// TestWSConnmgr_ConcurrentSubscribeUnsubscribePublish_NoRace stresses the
// broker-backed WSConnmgr with concurrent Subscribe, Unsubscribe, and
// Publish calls across overlapping exact and wildcard topics. It asserts no
// panic/deadlock; correctness of fan-out under this churn is covered by the
// deterministic tests above, this one is a data-race and stability guard
// (run with -race).
func TestWSConnmgr_ConcurrentSubscribeUnsubscribePublish_NoRace(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	topics := []string{"chat.1", "chat.2", "chat.3", "chat.*"}
	noop := func(*memorybroker.Message) {}

	const goroutines = 32
	const iterations = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(g int) {
			defer wg.Done()
			connID := "conn-" + topics[g%len(topics)] + "-" + string(rune('a'+g))
			for i := 0; i < iterations; i++ {
				topic := topics[i%len(topics)]
				if err := connmgr.Subscribe(connID, topic, noop); err != nil {
					t.Errorf("Subscribe: %v", err)
				}
				if err := (*connmgr.Broker).Publish(topic, "x"); err != nil {
					t.Errorf("Publish: %v", err)
				}
				if err := connmgr.Unsubscribe(connID, topic); err != nil {
					t.Errorf("Unsubscribe: %v", err)
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestStress_WSConnmgr_ConcurrentLifecycle(t *testing.T) {
	const workers = 12
	const iterations = 60

	sharedConn, cleanup := wslTestConn(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	connmgr.maxConns = 0

	var wg sync.WaitGroup

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				id := fmt.Sprintf("conn-%d-%d", w, i)

				c := connmgr.Register(id, sharedConn)
				if c == nil {
					continue
				}

				topic := fmt.Sprintf("topic.%d", i%4)
				_ = connmgr.Subscribe(id, topic, func(*memorybroker.Message) {})
				connmgr.isSubscribed(id, topic)
				connmgr.touch(id)
				connmgr.Count()
				connmgr.Get(id)
				_ = connmgr.Unsubscribe(id, topic)

				connmgr.Unregister(id)
				connmgr.Unregister(id)
			}
		}(w)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			connmgr.reapDeadConns(time.Nanosecond)
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()

	if got := connmgr.Count(); got != 0 {
		t.Errorf("every connection must be released after the churn, %d still tracked", got)
	}

	connmgr.mu.RLock()
	leftover := len(connmgr.subscriptions)
	connmgr.mu.RUnlock()
	if leftover != 0 {
		t.Errorf("every subscription list must be released after the churn, %d still tracked", leftover)
	}
}

func TestStress_WSConnmgr_MaxConnsEnforcedUnderConcurrency(t *testing.T) {
	sharedConn, cleanup := wslTestConn(t)
	defer cleanup()

	const limit = 10
	connmgr := NewWSConnmgr(log.NewLog(nil), nil)
	connmgr.maxConns = limit

	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted := 0

	for w := 0; w < 32; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			if c := connmgr.Register(fmt.Sprintf("c-%d", w), sharedConn); c != nil {
				mu.Lock()
				accepted++
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()

	if accepted != limit {
		t.Errorf("concurrent Register must admit exactly maxConns connections, admitted %d want %d", accepted, limit)
	}
	if got := connmgr.Count(); got != limit {
		t.Errorf("tracked connections must equal maxConns, got %d want %d", got, limit)
	}
}
