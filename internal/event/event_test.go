package event

import (
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestEvent_OnAndEmitInRegistrationOrder(t *testing.T) {
	e := NewEvent()
	var got []string
	e.On("x", func(args ...any) { got = append(got, "first:"+args[0].(string)) })
	e.On("x", func(args ...any) { got = append(got, "second:"+args[0].(string)) })
	e.On("y", func(args ...any) { got = append(got, "other") })

	e.Emit("x", "payload")

	want := []string{"first:payload", "second:payload"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Error(test.DiffMessage(got, want, "listeners run in order, only for their event"))
	}
}

func TestEvent_EmitWithNoListeners(t *testing.T) {
	e := NewEvent()
	e.Emit("nobody", 1, 2, 3)
}

func TestEvent_HasListeners(t *testing.T) {
	e := NewEvent()
	if e.HasListeners("x") {
		t.Error(test.DiffMessage(true, false, "new Event has no listeners"))
	}
	e.On("x", func(...any) {})
	if !e.HasListeners("x") {
		t.Error(test.DiffMessage(false, true, "after On"))
	}
	if e.HasListeners("y") {
		t.Error(test.DiffMessage(true, false, "other event name"))
	}
}

func TestEvent_ListenerPanicIsRecoveredAndOthersStillRun(t *testing.T) {
	e := NewEvent()
	ran := false
	e.On("x", func(...any) { panic("boom") })
	e.On("x", func(...any) { ran = true })

	output := captureStderr(t, func() { e.Emit("x") })

	if !ran {
		t.Error(test.DiffMessage(ran, true, "a panicking listener must not stop the next one"))
	}
	if output != "event: listener panic recovered: boom\n" {
		t.Error(test.DiffMessage(output, "event: listener panic recovered: boom\n", "panic is reported on stderr"))
	}
}

func TestEvent_WarnsAboveMaxListeners(t *testing.T) {
	e := NewEvent()
	output := captureStderr(t, func() {
		for range maxListenersPerEvent {
			e.On("x", func(...any) {})
		}
	})
	if output != "" {
		t.Error(test.DiffMessage(output, "", "no warning up to the limit"))
	}
	output = captureStderr(t, func() { e.On("x", func(...any) {}) })
	if output != "event: possible listener leak: 11 listeners on \"x\"\n" {
		t.Error(test.DiffMessage(output, "event: possible listener leak: 11 listeners on \"x\"\n", "warning above the limit"))
	}
}

func TestEvent_OnDuringEmitDoesNotAffectThatEmit(t *testing.T) {
	e := NewEvent()
	calls := 0
	e.On("x", func(...any) {
		calls++
		e.On("x", func(...any) { calls += 100 })
	})

	e.Emit("x")

	if calls != 1 {
		t.Error(test.DiffMessage(calls, 1, "Emit iterates the snapshot taken when it started"))
	}
}

func TestEvent_ConcurrentOnEmitHasListeners(t *testing.T) {
	e := NewEvent()
	names := []string{"a", "b", "c", "d"}
	var emitted atomic.Int64
	var wg sync.WaitGroup
	for _, name := range names {
		wg.Go(func() {
			for range maxListenersPerEvent {
				e.On(name, func(...any) { emitted.Add(1) })
			}
		})
	}
	for range 8 {
		wg.Go(func() {
			for i := range 1000 {
				name := names[i%len(names)]
				e.Emit(name, "payload")
				_ = e.HasListeners(name)
			}
		})
	}
	wg.Wait()

	for _, name := range names {
		if !e.HasListeners(name) {
			t.Error(test.DiffMessage(false, true, "listeners for "+name+" survive concurrent registration"))
		}
	}
	before := emitted.Load()
	e.Emit("a")
	if got := emitted.Load() - before; got != maxListenersPerEvent {
		t.Error(test.DiffMessage(got, int64(maxListenersPerEvent), "every concurrent On is kept"))
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stderr
	os.Stderr = writer
	done := make(chan string)
	go func() {
		data, _ := io.ReadAll(reader)
		done <- string(data)
	}()
	fn()
	os.Stderr = original
	_ = writer.Close()
	return <-done
}

func TestEvent_ZeroValueIsUsable(t *testing.T) {
	var e Event
	if e.HasListeners("x") {
		t.Error(test.DiffMessage(true, false, "zero Event has no listeners"))
	}
	e.Emit("x")
	called := false
	e.On("x", func(...any) { called = true })
	e.Emit("x")
	if !called {
		t.Error(test.DiffMessage(called, true, "zero Event accepts listeners"))
	}
}
