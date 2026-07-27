package core

import (
	"sync"
	"testing"
	"time"

	"golang.org/x/net/websocket"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
)

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

// TestWSConnmgr_ConcurrentSubscribeUnsubscribePublish_NoRace stresses the
// broker-backed WSConnmgr with concurrent Subscribe, Unsubscribe, and
// Publish calls across overlapping exact and wildcard topics. It asserts no
// panic/deadlock; correctness of fan-out under this churn is covered by the
// deterministic tests above, this one is a data-race and stability guard
// (run with -race).
func TestWSConnmgr_ConcurrentSubscribeUnsubscribePublish_NoRace(t *testing.T) {
	connmgr := NewWSConnmgr(log.NewLog(nil))
	topics := []string{"chat.1", "chat.2", "chat.3", "chat.*"}
	noop := func(*broker.Message) {}

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
				if err := connmgr.Broker.Publish(topic, "x"); err != nil {
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
