package connmgr

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/internal/test"
	"golang.org/x/net/websocket"
)

// newTestConn creates a Connection without a real websocket for tests that do
// not exercise the write loop.
func newTestConn(userID string) *Connection {
	id, err := crypto.UUID()
	if err != nil {
		panic(err)
	}
	return &Connection{
		ID:        id,
		UserID:    userID,
		CreatedAt: time.Now(),
		send:      make(chan []byte, sendBufferSize),
		done:      make(chan struct{}),
	}
}

// makeWSPair starts a minimal httptest WebSocket server and returns the
// server-side and client-side connections along with a cleanup function.
func makeWSPair(t *testing.T) (serverConn *websocket.Conn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()
	serverCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(websocket.Handler(func(ws *websocket.Conn) {
		serverCh <- ws
		buf := make([]byte, 1)
		_, _ = ws.Read(buf)
	}))

	origin := srv.URL
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/"
	client, err := websocket.Dial(url, "", origin)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}

	serverConn = <-serverCh
	return serverConn, client, func() {
		_ = client.Close()
		srv.Close()
	}
}

func TestNewConnection_Fields(t *testing.T) {
	serverConn, _, cleanup := makeWSPair(t)
	defer cleanup()

	before := time.Now()
	c, err := NewConnection(serverConn, "user1")
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	after := time.Now()

	if c.ID == "" {
		t.Error(test.DiffMessage(c.ID, "non-empty UUID", "ID"))
	}
	if c.UserID != "user1" {
		t.Error(test.DiffMessage(c.UserID, "user1", "UserID"))
	}
	if c.CreatedAt.Before(before) || c.CreatedAt.After(after) {
		t.Error(test.DiffMessage(c.CreatedAt, "within test window", "CreatedAt"))
	}
	if c.conn != serverConn {
		t.Error(test.DiffMessage(c.conn, serverConn, "conn"))
	}
}

func TestNewConnection_Anonymous(t *testing.T) {
	serverConn, _, cleanup := makeWSPair(t)
	defer cleanup()

	c, err := NewConnection(serverConn, "")
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	if c.UserID != "" {
		t.Error(test.DiffMessage(c.UserID, "", "anonymous connection must have empty UserID"))
	}
}

func TestNewConnection_UniqueIDs(t *testing.T) {
	serverConn, _, cleanup := makeWSPair(t)
	defer cleanup()

	a, err := NewConnection(serverConn, "")
	if err != nil {
		t.Fatalf("NewConnection a: %v", err)
	}
	b, err := NewConnection(serverConn, "")
	if err != nil {
		t.Fatalf("NewConnection b: %v", err)
	}
	if a.ID == b.ID {
		t.Error(test.DiffMessage(a.ID, "different from "+b.ID, "IDs must be unique"))
	}
}

func TestConnection_Conn(t *testing.T) {
	serverConn, _, cleanup := makeWSPair(t)
	defer cleanup()

	c, err := NewConnection(serverConn, "")
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	if c.Conn() != serverConn {
		t.Error(test.DiffMessage(c.Conn(), serverConn, "Conn()"))
	}
}

func TestConnection_Send_Happy(t *testing.T) {
	c := newTestConn("")
	if !c.Send([]byte("hello")) {
		t.Error(test.DiffMessage(false, true, "Send should succeed on open connection"))
	}
}

func TestConnection_Send_BufferFull(t *testing.T) {
	c := newTestConn("")
	for i := 0; i < sendBufferSize; i++ {
		c.Send([]byte("x"))
	}
	if c.Send([]byte("overflow")) {
		t.Error(test.DiffMessage(true, false, "Send must return false when buffer is full"))
	}
}

// TestConnection_Send_AfterClose_NoPanic verifies that Send never panics after
// Close, even when called many times. With the select-done pattern, Send is
// non-deterministic once done is closed (may return true or false depending on
// whether done or send is selected), but it must never panic.
func TestConnection_Send_AfterClose_NoPanic(t *testing.T) {
	c := newTestConn("")
	c.Close()
	for i := 0; i < sendBufferSize*2; i++ {
		c.Send([]byte("msg"))
	}
}

// TestConnection_Send_FullBufferAndClosed verifies that once the send buffer is
// full AND done is closed, Send always returns false. This is the deterministic
// case: only the done case is selectable.
func TestConnection_Send_FullBufferAndClosed(t *testing.T) {
	c := newTestConn("")
	for i := 0; i < sendBufferSize; i++ {
		c.Send([]byte("x"))
	}
	c.Close()
	if c.Send([]byte("msg")) {
		t.Error(test.DiffMessage(true, false, "Send must return false when buffer full and connection closed"))
	}
}

func TestConnection_Send_EmptyPayload(t *testing.T) {
	c := newTestConn("")
	if !c.Send([]byte{}) {
		t.Error(test.DiffMessage(false, true, "Send empty slice should succeed"))
	}
}

func TestConnection_Close_Idempotent(t *testing.T) {
	c := newTestConn("")
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Close panicked on second call: %v", r)
		}
	}()
	c.Close()
	c.Close()
	c.Close()
}

func TestConnection_Close_SetsDone(t *testing.T) {
	c := newTestConn("")
	c.Close()
	select {
	case <-c.Done():
	default:
		t.Error(test.DiffMessage("open", "closed", "Done() channel must be closed after Close()"))
	}
}

func TestConnection_Done_BlocksBeforeClose(t *testing.T) {
	c := newTestConn("")
	select {
	case <-c.Done():
		t.Error(test.DiffMessage("closed", "open", "Done() must not be closed before Close()"))
	default:
	}
}

func TestConnection_Start_WritesMessage(t *testing.T) {
	serverConn, clientConn, cleanup := makeWSPair(t)
	defer cleanup()

	c, err := NewConnection(serverConn, "")
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c.Start(ctx)

	payload := []byte(`{"event":"test","payload":{}}`)
	if !c.Send(payload) {
		t.Fatal("Send returned false unexpectedly")
	}

	_ = clientConn.SetDeadline(time.Now().Add(2 * time.Second))
	var received []byte
	if err := websocket.Message.Receive(clientConn, &received); err != nil {
		t.Fatalf("client receive: %v", err)
	}
	if string(received) != string(payload) {
		t.Error(test.DiffMessage(string(received), string(payload), "received message"))
	}
}

func TestConnection_Start_ContextCancel(t *testing.T) {
	serverConn, _, cleanup := makeWSPair(t)
	defer cleanup()

	c, err := NewConnection(serverConn, "")
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.Start(ctx)

	cancel()

	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Error("connection was not closed after context cancellation")
	}
}

func TestConnection_Start_Idempotent(t *testing.T) {
	serverConn, clientConn, cleanup := makeWSPair(t)
	defer cleanup()

	c, err := NewConnection(serverConn, "")
	if err != nil {
		t.Fatalf("NewConnection: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c.Start(ctx)
	c.Start(ctx)
	c.Start(ctx)

	payload := []byte("once")
	c.Send(payload)

	_ = clientConn.SetDeadline(time.Now().Add(time.Second))
	var got []byte
	if err := websocket.Message.Receive(clientConn, &got); err != nil {
		t.Fatalf("receive: %v", err)
	}

	_ = clientConn.SetDeadline(time.Now().Add(50 * time.Millisecond))
	var extra []byte
	if err := websocket.Message.Receive(clientConn, &extra); err == nil {
		t.Error(test.DiffMessage(string(extra), "no second message", "Start must not spawn multiple write loops"))
	}
}

func TestConnection_Concurrent_Send(t *testing.T) {
	c := newTestConn("")
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Send([]byte("concurrent"))
		}()
	}
	wg.Wait()
}

func TestConnection_Concurrent_Close(t *testing.T) {
	c := newTestConn("")
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Close()
		}()
	}
	wg.Wait()
	select {
	case <-c.Done():
	default:
		t.Error(test.DiffMessage("open", "closed", "Done must be closed after concurrent Close calls"))
	}
}

func TestConnection_Concurrent_SendAndClose(t *testing.T) {
	c := newTestConn("")
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.Send([]byte("msg"))
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.Close()
	}()
	wg.Wait()
}
