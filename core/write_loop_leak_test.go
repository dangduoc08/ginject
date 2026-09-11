package core

import (
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
)

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
