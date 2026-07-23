package event

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestEvent_OnAndEmit(t *testing.T) {
	e := NewEvent()
	var got []any
	e.On("greet", func(args ...any) {
		got = append(got, args...)
	})
	e.Emit("greet", "hello")
	e.Emit("greet", "world")

	if len(got) != 2 || got[0] != "hello" || got[1] != "world" {
		t.Error(test.DiffMessage(got, []any{"hello", "world"}, "On listener should fire on every Emit"))
	}
}

func TestEvent_OnceFiresOnlyOnce(t *testing.T) {
	e := NewEvent()
	var n int32
	e.Once("done", func(args ...any) {
		atomic.AddInt32(&n, 1)
	})
	e.Emit("done")
	e.Emit("done")

	if n != 1 {
		t.Error(test.DiffMessage(n, int32(1), "Once listener should fire exactly once"))
	}
}

func TestEvent_EmitWithNoListeners(t *testing.T) {
	e := NewEvent()
	e.Emit("nobody-listening")
}

func TestEvent_Off(t *testing.T) {
	e := NewEvent()
	var n int32
	fn := func(args ...any) { atomic.AddInt32(&n, 1) }
	e.On("x", fn)
	e.Off("x", fn)
	e.Emit("x")

	if n != 0 {
		t.Error(test.DiffMessage(n, int32(0), "Off should remove the listener before Emit"))
	}
}

func TestEvent_OffOnceListener(t *testing.T) {
	e := NewEvent()
	var n int32
	fn := func(args ...any) { atomic.AddInt32(&n, 1) }
	e.Once("x", fn)
	e.Off("x", fn)
	e.Emit("x")

	if n != 0 {
		t.Error(test.DiffMessage(n, int32(0), "Off should remove a once listener before Emit"))
	}
}

func TestEvent_RemoveAllListeners(t *testing.T) {
	e := NewEvent()
	var n int32
	e.On("x", func(args ...any) { atomic.AddInt32(&n, 1) })
	e.Once("x", func(args ...any) { atomic.AddInt32(&n, 1) })
	e.RemoveAllListeners("x")
	e.Emit("x")

	if n != 0 {
		t.Error(test.DiffMessage(n, int32(0), "RemoveAllListeners should clear both On and Once listeners"))
	}
}

func TestEvent_ListenerCount(t *testing.T) {
	e := NewEvent()
	e.On("x", func(args ...any) {})
	e.On("x", func(args ...any) {})
	e.Once("x", func(args ...any) {})

	if n := e.ListenerCount("x"); n != 3 {
		t.Error(test.DiffMessage(n, 3, "ListenerCount should count On and Once listeners"))
	}
	if n := e.ListenerCount("unknown"); n != 0 {
		t.Error(test.DiffMessage(n, 0, "ListenerCount for unknown event should be 0"))
	}
}

func TestEvent_HasListeners(t *testing.T) {
	e := NewEvent()
	if e.HasListeners("x") {
		t.Error(test.DiffMessage(true, false, "HasListeners should be false before any listener is added"))
	}
	e.On("x", func(args ...any) {})
	if !e.HasListeners("x") {
		t.Error(test.DiffMessage(false, true, "HasListeners should be true after On"))
	}
}

func TestEvent_EventNames(t *testing.T) {
	e := NewEvent()
	e.On("a", func(args ...any) {})
	e.Once("b", func(args ...any) {})

	names := e.EventNames()
	seen := map[string]bool{}
	for _, n := range names {
		seen[n] = true
	}
	if !seen["a"] || !seen["b"] || len(names) != 2 {
		t.Error(test.DiffMessage(names, []string{"a", "b"}, "EventNames should list all registered event names"))
	}
}

func TestEvent_ListenerPanicIsRecovered(t *testing.T) {
	e := NewEvent()
	var afterPanicRan bool
	e.On("x", func(args ...any) { panic("boom") })
	e.On("x", func(args ...any) { afterPanicRan = true })

	e.Emit("x")

	if !afterPanicRan {
		t.Error(test.DiffMessage(afterPanicRan, true, "a panicking listener should not stop later listeners from running"))
	}
}

func TestEvent_SetMaxListenersSuppressesWarning(t *testing.T) {
	e := NewEvent()
	e.SetMaxListeners(1)
	e.On("x", func(args ...any) {})
	e.On("x", func(args ...any) {})
}

func TestEvent_Reset(t *testing.T) {
	e := NewEvent()
	e.On("x", func(args ...any) {})
	e.Once("y", func(args ...any) {})
	e.Reset()

	if e.HasListeners("x") || e.HasListeners("y") {
		t.Error(test.DiffMessage(true, false, "reset should clear all listeners"))
	}
}

func TestEvent_ConcurrentOnOffEmit_NoDataRace(t *testing.T) {
	e := NewEvent()
	const goroutines = 16
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			e.On("x", func(args ...any) {})
		}()
		go func() {
			defer wg.Done()
			e.Emit("x")
		}()
		go func() {
			defer wg.Done()
			e.ListenerCount("x")
		}()
	}

	wg.Wait()
}
