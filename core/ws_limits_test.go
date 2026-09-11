package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
	"golang.org/x/net/websocket"
)

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
