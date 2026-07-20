package core

import (
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/dangduoc08/ginject/broker2"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
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

	connmgr := NewWSConnmgr(log.NewLog(nil))
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

	connmgr := NewWSConnmgr(log.NewLog(nil))
	conn := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	const goroutines = 16
	const perGoroutine = 8
	total := goroutines * perGoroutine

	// TrySend is non-blocking and may legitimately drop payloads under a
	// burst this size (sendBufferSize is 32) — that's the contract, not a
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

	connmgr := NewWSConnmgr(log.NewLog(nil))
	// No client-side reads happen in this test, so once the writer goroutine
	// blocks on its own in-flight websocket.JSON.Send, the buffer fills up
	// and TrySend must start returning false instead of blocking forever.
	conn := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	done := make(chan bool)
	go func() {
		sawDrop := false
		for i := 0; i < sendBufferSize*4; i++ {
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

	connmgr := NewWSConnmgr(log.NewLog(nil))
	conn := connmgr.Register("conn-1", serverConn)

	connmgr.Unregister("conn-1")

	// A send racing with (or arriving after) Unregister must not panic —
	// done is closed, not send, precisely so this stays safe.
	conn.TrySend(WSPayload{Type: TypeEvent, Message: "after-unregister"})
}

func TestWSConnmgr_Get(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	connmgr := NewWSConnmgr(log.NewLog(nil))
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

	connmgr := NewWSConnmgr(log.NewLog(nil))
	conn := connmgr.Register("conn-1", serverConn)
	defer connmgr.Unregister("conn-1")

	before := conn.LastSeen
	time.Sleep(time.Millisecond)
	connmgr.touch("conn-1")

	got, _ := connmgr.Get("conn-1")
	if !got.LastSeen.After(before) {
		t.Error(test.DiffMessage(got.LastSeen, "after "+before.String(), "touch should advance LastSeen"))
	}
}

func TestWSConnmgr_TouchUnknownConnIsNoop(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil))
	connmgr.touch("missing")
}

func TestWSConnmgr_UnsubscribeRemovesOnlyMatchingTopic(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil))

	if err := connmgr.Subscribe("conn-1", "topic.a", func(*broker2.Message) {}); err != nil {
		t.Fatal(err)
	}
	if err := connmgr.Subscribe("conn-1", "topic.b", func(*broker2.Message) {}); err != nil {
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
	connmgr := NewWSConnmgr(log.NewLog(nil))
	if err := connmgr.Unsubscribe("conn-1", "never-subscribed"); err != nil {
		t.Error(test.DiffMessage(err, nil, "Unsubscribe on a topic never subscribed to should not error"))
	}
}
