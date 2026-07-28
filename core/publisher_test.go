package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
	"golang.org/x/net/websocket"
)

func TestKnownHTTPDependencyKeys_IncludesPublisherKey(t *testing.T) {
	if _, ok := knownHTTPDependencyKeys[publisherKey]; !ok {
		t.Error(test.DiffMessage(ok, true, "knownHTTPDependencyKeys should include publisherKey"))
	}
}

func TestKnownWSDependencyKeys_IncludesPublisherKey(t *testing.T) {
	if _, ok := knownWSDependencyKeys[publisherKey]; !ok {
		t.Error(test.DiffMessage(ok, true, "knownWSDependencyKeys should include publisherKey"))
	}
}

func TestGetHTTPDependency_ResolvesPublisher(t *testing.T) {
	app := New()

	got := getHTTPDependency(publisherKey, nil, reflect.Value{})
	p, ok := got.(common.Publisher)
	if !ok {
		t.Fatalf("expected common.Publisher, got %T", got)
	}
	if err := p.Publish("some.topic"); err != nil {
		t.Error(test.DiffMessage(err, nil, "Publish through the resolved dependency should not error"))
	}
	_ = app
}

func TestGetWSDependency_ResolvesPublisher(t *testing.T) {
	app := New()

	got := getWSDependency(publisherKey, nil, reflect.Value{})
	p, ok := got.(common.Publisher)
	if !ok {
		t.Fatalf("expected common.Publisher, got %T", got)
	}
	if err := p.Publish("some.topic"); err != nil {
		t.Error(test.DiffMessage(err, nil, "Publish through the resolved dependency should not error"))
	}
	_ = app
}

func TestPublisher_SharesBrokerWithWSConnmgr(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.EnableWS(&WSConfig{Path: "/ws"})
	app.Create(ModuleBuilder().Build())

	if *app.ws.connmgr.Broker != app.broker {
		t.Error(test.DiffMessage(*app.ws.connmgr.Broker, app.broker, "WSConnmgr must use the same broker instance as the app's Publisher, or fan-out across transports won't reach WS subscribers"))
	}
}

type publisherTestController struct {
	common.HTTP
	common.WS
}

func (c publisherTestController) NewController() Controller { return c }

func (c publisherTestController) READ_ping(publisher common.Publisher) ctx.Map {
	_ = publisher.Publish("chat.123")
	return ctx.Map{"message": "ok"}
}

func (c publisherTestController) SUBSCRIBE_chat_ANY() ctx.Map {
	return ctx.Map{"ok": true}
}

func TestPublisher_HTTPHandlerFansOutToWSSubscriber(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.EnableWS(&WSConfig{Path: "/ws"})
	app.Create(ModuleBuilder().Controllers(publisherTestController{}).Build())

	server := httptest.NewServer(app)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	var connected WSPayload
	if err := websocket.JSON.Receive(conn, &connected); err != nil {
		t.Fatalf("receive connected: %v", err)
	}

	if err := websocket.JSON.Send(conn, WSPayload{Type: TypeSubscribe, ID: "1", Topic: []string{"chat.123"}}); err != nil {
		t.Fatalf("send subscribe: %v", err)
	}
	var ack WSPayload
	if err := websocket.JSON.Receive(conn, &ack); err != nil {
		t.Fatalf("receive ack: %v", err)
	}
	if ack.Type != TypeAck {
		t.Fatalf("expected ack, got %v", ack.Type)
	}

	r := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var evt WSPayload
	if err := websocket.JSON.Receive(conn, &evt); err != nil {
		t.Fatalf("receive event: %v", err)
	}
	if evt.Type != TypeEvent || len(evt.Topic) != 1 || evt.Topic[0] != "chat.123" {
		t.Error(test.DiffMessage(evt, "event chat.123", "the HTTP handler's Publish call should fan out to the WS subscriber"))
	}
}

type publisherWSTestController struct {
	common.WS
}

func (c publisherWSTestController) NewController() Controller { return c }

func (c publisherWSTestController) SUBSCRIBE_relay_ANY(publisher common.Publisher) ctx.Map {
	_ = publisher.Publish("relay.out")
	return ctx.Map{"ok": true}
}

func TestPublisher_WSHandlerFansOutToOtherWSSubscriber(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.EnableWS(&WSConfig{Path: "/ws"})
	app.Create(ModuleBuilder().Controllers(publisherWSTestController{}).Build())

	server := httptest.NewServer(app)
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dial := func() *websocket.Conn {
		t.Helper()
		conn, err := websocket.Dial(wsURL, "", server.URL)
		if err != nil {
			t.Fatalf("dial: %v", err)
		}
		var connected WSPayload
		if err := websocket.JSON.Receive(conn, &connected); err != nil {
			t.Fatalf("receive connected: %v", err)
		}
		return conn
	}
	subscribe := func(conn *websocket.Conn, topic string) {
		t.Helper()
		if err := websocket.JSON.Send(conn, WSPayload{Type: TypeSubscribe, ID: "sub-" + topic, Topic: []string{topic}}); err != nil {
			t.Fatalf("send subscribe: %v", err)
		}
		var ack WSPayload
		if err := websocket.JSON.Receive(conn, &ack); err != nil {
			t.Fatalf("receive ack: %v", err)
		}
		if ack.Type != TypeAck {
			t.Fatalf("expected ack for %s, got %v", topic, ack.Type)
		}
	}

	relayConn := dial()
	defer func() { _ = relayConn.Close() }()
	subscribe(relayConn, "relay.in")

	outConn := dial()
	defer func() { _ = outConn.Close() }()
	subscribe(outConn, "relay.out")

	if err := websocket.JSON.Send(relayConn, WSPayload{Type: TypePublish, ID: "pub-1", Topic: []string{"relay.in"}, Message: "hi"}); err != nil {
		t.Fatalf("send publish: %v", err)
	}

	var outEvt WSPayload
	if err := websocket.JSON.Receive(outConn, &outEvt); err != nil {
		t.Fatalf("receive event on relay.out: %v", err)
	}
	if outEvt.Type != TypeEvent || len(outEvt.Topic) != 1 || outEvt.Topic[0] != "relay.out" {
		t.Error(test.DiffMessage(outEvt, "event relay.out", "the WS handler's injected Publisher should fan out to a subscriber of a different topic"))
	}
}
