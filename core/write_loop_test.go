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
	go writeLoop(serverConn, send, done, DefaultWSWriteTimeout, log.NewLog(nil))

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
		writeLoop(serverConn, send, done, DefaultWSWriteTimeout, log.NewLog(nil))
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
		writeLoop(serverConn, send, done, DefaultWSWriteTimeout, log.NewLog(nil))
		close(finished)
	}()

	send <- WSPayload{Type: TypeEvent, Message: "hi"}

	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("writeLoop should return once the send fails because the client closed the connection")
	}
}

func TestWriteLoop_WriteFailure_ClosesConnAndExits(t *testing.T) {
	serverConn, cleanup := wslTestConn(t)
	defer cleanup()

	send := make(chan WSPayload, 1)
	done := make(chan struct{})
	defer close(done)

	exited := make(chan struct{})
	go func() {
		writeLoop(serverConn, send, done, DefaultWSWriteTimeout, log.NewLog(nil))
		close(exited)
	}()

	_ = serverConn.Close()
	send <- WSPayload{Type: TypeEvent, Message: "after close"}

	select {
	case <-exited:
	case <-time.After(3 * time.Second):
		t.Fatal(test.DiffMessage("still running", "exited", "writeLoop must return once a write fails instead of spinning on a dead socket"))
	}
}

func TestWriteLoop_StalledPeer_HonorsWriteDeadline(t *testing.T) {
	serverConn, cleanup := wslTestConn(t)
	defer cleanup()

	send := make(chan WSPayload, 256)
	done := make(chan struct{})
	defer close(done)

	exited := make(chan struct{})
	go func() {
		writeLoop(serverConn, send, done, 200*time.Millisecond, log.NewLog(nil))
		close(exited)
	}()

	big := make([]byte, 1<<20)
	for i := range big {
		big[i] = 'x'
	}
	for i := 0; i < cap(send); i++ {
		send <- WSPayload{Type: TypeEvent, Message: string(big)}
	}

	select {
	case <-exited:
	case <-time.After(15 * time.Second):
		t.Fatal(test.DiffMessage("blocked forever", "exited", "a peer that never reads must trip the write deadline, not pin the write goroutine"))
	}
}
