package core

import (
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
)

func TestWriteLoop_SendsPayloadToConn(t *testing.T) {
	serverConn, clientConn, cleanup := newTestWSConnPair(t)
	defer cleanup()

	send := make(chan WSPayload, 1)
	done := make(chan struct{})
	go writeLoop(serverConn, send, done, log.NewLog(nil))

	send <- WSPayload{Type: TypeEvent, Message: "hi"}
	got := recvWSPayload(t, clientConn)

	if got.Type != TypeEvent || got.Message != "hi" {
		t.Error(test.DiffMessage(got, WSPayload{Type: TypeEvent, Message: "hi"}, "writeLoop should forward payloads sent on the send channel"))
	}

	close(done)
}

func TestWriteLoop_StopsOnDone(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()

	send := make(chan WSPayload)
	done := make(chan struct{})

	finished := make(chan struct{})
	go func() {
		writeLoop(serverConn, send, done, log.NewLog(nil))
		close(finished)
	}()

	close(done)

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("writeLoop should return once done is closed")
	}
}

func TestWriteLoop_StopsOnSendError(t *testing.T) {
	serverConn, _, cleanup := newTestWSConnPair(t)
	defer cleanup()
	_ = serverConn.Close()

	send := make(chan WSPayload, 1)
	done := make(chan struct{})
	defer close(done)

	finished := make(chan struct{})
	go func() {
		writeLoop(serverConn, send, done, log.NewLog(nil))
		close(finished)
	}()

	send <- WSPayload{Type: TypeEvent, Message: "hi"}

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("writeLoop should return once the send fails because the client closed the connection")
	}
}
