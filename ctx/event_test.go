package ctx

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/dangduoc08/ginject/testutils"
)

// --- On ---

func TestOn_HappyPath(t *testing.T) {
	e := NewEvent()
	called := 0
	e.On("test", func(args ...any) { called++ })
	e.Emit("test")
	if called != 1 {
		t.Error(testutils.DiffMessage(called, 1, "On: single listener should fire once"))
	}
}

func TestOn_MultipleListeners(t *testing.T) {
	e := NewEvent()
	calls := 0
	e.On("test", func(args ...any) { calls++ })
	e.On("test", func(args ...any) { calls++ })
	e.On("test", func(args ...any) { calls++ })
	e.Emit("test")
	if calls != 3 {
		t.Error(testutils.DiffMessage(calls, 3, "On: all listeners should fire"))
	}
}

func TestOn_ListenerOrder(t *testing.T) {
	e := NewEvent()
	var order []int
	e.On("test", func(args ...any) { order = append(order, 1) })
	e.On("test", func(args ...any) { order = append(order, 2) })
	e.On("test", func(args ...any) { order = append(order, 3) })
	e.Emit("test")
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Error(testutils.DiffMessage(order, []int{1, 2, 3}, "On: listener order must be stable"))
	}
}

func TestOn_NoListeners_NoEmit(t *testing.T) {
	e := NewEvent()
	e.Emit("noevent") // must not panic
}

func TestOn_EmptyEventName(t *testing.T) {
	e := NewEvent()
	called := 0
	e.On("", func(args ...any) { called++ })
	e.Emit("")
	if called != 1 {
		t.Error(testutils.DiffMessage(called, 1, "On: empty event name is a valid key"))
	}
}

func TestOn_ArgsPassedThrough(t *testing.T) {
	e := NewEvent()
	var got []any
	e.On("test", func(args ...any) { got = args })
	e.Emit("test", "hello", 42)
	if len(got) != 2 || got[0] != "hello" || got[1] != 42 {
		t.Error(testutils.DiffMessage(got, []any{"hello", 42}, "On: args should be forwarded to listener"))
	}
}

// --- Once ---

func TestOnce_HappyPath(t *testing.T) {
	e := NewEvent()
	called := 0
	e.Once("test", func(args ...any) { called++ })
	e.Emit("test")
	e.Emit("test")
	if called != 1 {
		t.Error(testutils.DiffMessage(called, 1, "Once: should fire exactly once"))
	}
}

func TestOnce_MultipleListeners(t *testing.T) {
	e := NewEvent()
	calls := 0
	e.Once("test", func(args ...any) { calls++ })
	e.Once("test", func(args ...any) { calls++ })
	e.Emit("test")
	if calls != 2 {
		t.Error(testutils.DiffMessage(calls, 2, "Once: all once-listeners fire on first Emit"))
	}
	calls = 0
	e.Emit("test")
	if calls != 0 {
		t.Error(testutils.DiffMessage(calls, 0, "Once: no once-listeners should fire on second Emit"))
	}
}

func TestOnce_StrictSemanticsUnderConcurrentEmit(t *testing.T) {
	e := NewEvent()
	var count atomic.Int64
	e.Once("test", func(args ...any) { count.Add(1) })

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Emit("test")
		}()
	}
	wg.Wait()

	if count.Load() != 1 {
		t.Error(testutils.DiffMessage(count.Load(), int64(1), "Once: must fire exactly once under 100 concurrent Emit calls"))
	}
}

func TestOnce_NewListenerAddedInsideCallback_NotLost(t *testing.T) {
	e := NewEvent()
	called := 0
	e.Once("test", func(args ...any) {
		// listener added inside callback must survive and fire on the next Emit
		e.Once("test", func(args ...any) { called++ })
	})
	e.Emit("test")
	e.Emit("test")
	if called != 1 {
		t.Error(testutils.DiffMessage(called, 1, "Once listener added inside callback must not be lost"))
	}
}

// --- Off ---

func TestOff_RemovesOnListener(t *testing.T) {
	e := NewEvent()
	calls := 0
	fn := func(args ...any) { calls++ }
	e.On("test", fn)
	e.Off("test", fn)
	e.Emit("test")
	if calls != 0 {
		t.Error(testutils.DiffMessage(calls, 0, "Off: removed On listener must not fire"))
	}
}

func TestOff_RemovesOnceListener(t *testing.T) {
	e := NewEvent()
	calls := 0
	fn := func(args ...any) { calls++ }
	e.Once("test", fn)
	e.Off("test", fn)
	e.Emit("test")
	if calls != 0 {
		t.Error(testutils.DiffMessage(calls, 0, "Off: removed Once listener must not fire"))
	}
}

func TestOff_RemovesOnlyMatchingListener(t *testing.T) {
	e := NewEvent()
	calls := 0
	fn1 := func(args ...any) { calls++ }
	fn2 := func(args ...any) { calls += 10 }
	e.On("test", fn1)
	e.On("test", fn2)
	e.Off("test", fn1)
	e.Emit("test")
	if calls != 10 {
		t.Error(testutils.DiffMessage(calls, 10, "Off: only the specified listener should be removed"))
	}
}

func TestOff_RemovesFirstOccurrenceOnly(t *testing.T) {
	e := NewEvent()
	calls := 0
	fn := func(args ...any) { calls++ }
	e.On("test", fn)
	e.On("test", fn)
	e.Off("test", fn)
	e.Emit("test")
	if calls != 1 {
		t.Error(testutils.DiffMessage(calls, 1, "Off: should remove only the first occurrence"))
	}
}

func TestOff_PreservesOrder(t *testing.T) {
	e := NewEvent()
	var order []int
	fn1 := func(args ...any) { order = append(order, 1) }
	fn2 := func(args ...any) { order = append(order, 2) }
	fn3 := func(args ...any) { order = append(order, 3) }
	e.On("test", fn1)
	e.On("test", fn2)
	e.On("test", fn3)
	e.Off("test", fn2)
	e.Emit("test")
	if len(order) != 2 || order[0] != 1 || order[1] != 3 {
		t.Error(testutils.DiffMessage(order, []int{1, 3}, "Off: listener order must remain stable after removal"))
	}
}

func TestOff_NonExistentListener_NoOp(t *testing.T) {
	e := NewEvent()
	fn := func(args ...any) {}
	e.Off("test", fn) // must not panic
}

func TestOff_NonExistentEvent_NoOp(t *testing.T) {
	e := NewEvent()
	fn := func(args ...any) {}
	e.Off("doesnotexist", fn) // must not panic
}

// --- RemoveAllListeners ---

func TestRemoveAllListeners_ClearsAll(t *testing.T) {
	e := NewEvent()
	calls := 0
	e.On("test", func(args ...any) { calls++ })
	e.Once("test", func(args ...any) { calls++ })
	e.RemoveAllListeners("test")
	e.Emit("test")
	if calls != 0 {
		t.Error(testutils.DiffMessage(calls, 0, "RemoveAllListeners: no listeners should fire"))
	}
}

func TestRemoveAllListeners_OnlyTargetEvent(t *testing.T) {
	e := NewEvent()
	calls := 0
	e.On("a", func(args ...any) { calls++ })
	e.On("b", func(args ...any) { calls += 10 })
	e.RemoveAllListeners("a")
	e.Emit("a")
	e.Emit("b")
	if calls != 10 {
		t.Error(testutils.DiffMessage(calls, 10, "RemoveAllListeners: must only clear the named event"))
	}
}

// --- SetMaxListeners ---

func TestSetMaxListeners_ZeroMeansUnlimited(t *testing.T) {
	e := NewEvent()
	e.SetMaxListeners(0)
	for i := 0; i < 20; i++ {
		e.On("test", func(args ...any) {})
	}
}

func TestSetMaxListeners_PositiveLimit(t *testing.T) {
	e := NewEvent()
	e.SetMaxListeners(2)
	e.On("test", func(args ...any) {})
	e.Once("test", func(args ...any) {})
	// combined = 2; third listener warns to stderr, no panic
	e.On("test", func(args ...any) {})
}

func TestSetMaxListeners_OnAndOnceCombined(t *testing.T) {
	e := NewEvent()
	e.SetMaxListeners(3)
	e.On("test", func(args ...any) {})
	e.On("test", func(args ...any) {})
	e.Once("test", func(args ...any) {})
	// combined = 3, no warning yet; 4th should warn
	e.Once("test", func(args ...any) {})
}

func TestSetMaxListeners_Default(t *testing.T) {
	e := NewEvent()
	if e.maxListeners != defaultMaxListeners {
		t.Error(testutils.DiffMessage(e.maxListeners, defaultMaxListeners, "default maxListeners should be 10"))
	}
}

// --- reset ---

func TestReset_ClearsListeners(t *testing.T) {
	e := NewEvent()
	calls := 0
	e.On("test", func(args ...any) { calls++ })
	e.Once("test", func(args ...any) { calls++ })
	e.reset()
	e.Emit("test")
	if calls != 0 {
		t.Error(testutils.DiffMessage(calls, 0, "reset: must clear all listeners"))
	}
}

func TestReset_PreservesMaxListeners(t *testing.T) {
	e := NewEvent()
	e.SetMaxListeners(5)
	e.reset()
	if e.maxListeners != 5 {
		t.Error(testutils.DiffMessage(e.maxListeners, 5, "reset: maxListeners must not be cleared"))
	}
}

// --- Emit: panic recovery ---

func TestEmit_PanicInListenerDoesNotAbortDispatch(t *testing.T) {
	e := NewEvent()
	secondCalled := false
	e.On("test", func(args ...any) { panic("oops") })
	e.On("test", func(args ...any) { secondCalled = true })
	e.Emit("test")
	if !secondCalled {
		t.Error(testutils.DiffMessage(secondCalled, true, "panic in first listener must not abort second listener"))
	}
}

func TestEmit_PanicInOnceListenerDoesNotAbortDispatch(t *testing.T) {
	e := NewEvent()
	secondCalled := false
	e.Once("test", func(args ...any) { panic("oops") })
	e.Once("test", func(args ...any) { secondCalled = true })
	e.Emit("test")
	if !secondCalled {
		t.Error(testutils.DiffMessage(secondCalled, true, "panic in first once-listener must not abort second"))
	}
}

// --- ListenerCount ---

func TestListenerCount_Empty(t *testing.T) {
	e := NewEvent()
	if n := e.ListenerCount("noop"); n != 0 {
		t.Error(testutils.DiffMessage(n, 0, "ListenerCount: unknown event should return 0"))
	}
}

func TestListenerCount_OnAndOnce(t *testing.T) {
	e := NewEvent()
	e.On("test", func(args ...any) {})
	e.On("test", func(args ...any) {})
	e.Once("test", func(args ...any) {})
	if n := e.ListenerCount("test"); n != 3 {
		t.Error(testutils.DiffMessage(n, 3, "ListenerCount must sum On and Once listeners"))
	}
}

func TestListenerCount_DecreasesAfterEmit(t *testing.T) {
	e := NewEvent()
	e.On("test", func(args ...any) {})
	e.Once("test", func(args ...any) {})
	e.Emit("test")
	if n := e.ListenerCount("test"); n != 1 {
		t.Error(testutils.DiffMessage(n, 1, "ListenerCount: once listener consumed by Emit should reduce count"))
	}
}

// --- HasListeners ---

func TestHasListeners_False_Empty(t *testing.T) {
	e := NewEvent()
	if e.HasListeners("noop") {
		t.Error(testutils.DiffMessage(true, false, "HasListeners: must return false for unknown event"))
	}
}

func TestHasListeners_True_AfterOn(t *testing.T) {
	e := NewEvent()
	e.On("test", func(args ...any) {})
	if !e.HasListeners("test") {
		t.Error(testutils.DiffMessage(false, true, "HasListeners: must return true after On"))
	}
}

func TestHasListeners_FalseAfterOnceConsumed(t *testing.T) {
	e := NewEvent()
	e.Once("test", func(args ...any) {})
	e.Emit("test")
	if e.HasListeners("test") {
		t.Error(testutils.DiffMessage(true, false, "HasListeners: must return false after once listener is consumed"))
	}
}

// --- EventNames ---

func TestEventNames_Empty(t *testing.T) {
	e := NewEvent()
	if names := e.EventNames(); len(names) != 0 {
		t.Error(testutils.DiffMessage(len(names), 0, "EventNames: must be empty for new event"))
	}
}

func TestEventNames_NoDuplicates(t *testing.T) {
	e := NewEvent()
	e.On("alpha", func(args ...any) {})
	e.Once("alpha", func(args ...any) {})
	e.On("beta", func(args ...any) {})
	names := e.EventNames()
	if len(names) != 2 {
		t.Error(testutils.DiffMessage(len(names), 2, "EventNames: must deduplicate event names"))
	}
}

func TestEventNames_ExcludesEmptied(t *testing.T) {
	e := NewEvent()
	fn := func(args ...any) {}
	e.On("test", fn)
	e.Off("test", fn)
	for _, name := range e.EventNames() {
		if name == "test" {
			t.Error(testutils.DiffMessage("test", "<absent>", "EventNames: must not include event with no listeners"))
		}
	}
}

// --- Concurrent access ---

func TestEvent_ConcurrentOnEmit(t *testing.T) {
	e := NewEvent()
	var mu sync.Mutex
	calls := 0

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.On("test", func(args ...any) {
				mu.Lock()
				calls++
				mu.Unlock()
			})
			e.Emit("test")
		}()
	}
	wg.Wait()
}

func TestEvent_ConcurrentOffEmit(t *testing.T) {
	e := NewEvent()
	var calls atomic.Int64
	fn := func(args ...any) { calls.Add(1) }
	e.On("test", fn)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			e.Emit("test")
		}()
		go func() {
			defer wg.Done()
			e.Off("test", fn)
		}()
	}
	wg.Wait()
}
