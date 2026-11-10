package event

import (
	"fmt"
	"maps"
	"os"
	"sync"
	"sync/atomic"
)

const maxListenersPerEvent = 10

type listenerMap map[string][]func(args ...any)

type Event struct {
	writeMu   sync.Mutex
	listeners atomic.Pointer[listenerMap]
}

func NewEvent() *Event {
	return &Event{}
}

func (e *Event) On(eventName string, fn func(args ...any)) {
	e.writeMu.Lock()
	current := e.snapshot()
	next := make(listenerMap, len(current)+1)
	maps.Copy(next, current)
	next[eventName] = append(append([]func(args ...any){}, current[eventName]...), fn)
	e.listeners.Store(&next)
	count := len(next[eventName])
	e.writeMu.Unlock()

	if count > maxListenersPerEvent {
		fmt.Fprintf(os.Stderr, "event: possible listener leak: %d listeners on %q\n", count, eventName)
	}
}

func (e *Event) Emit(eventName string, args ...any) {
	for _, listener := range e.snapshot()[eventName] {
		callEventListenerSafe(listener, args)
	}
}

func (e *Event) HasListeners(eventName string) bool {
	return len(e.snapshot()[eventName]) > 0
}

func (e *Event) snapshot() listenerMap {
	if current := e.listeners.Load(); current != nil {
		return *current
	}
	return nil
}

func callEventListenerSafe(listener func(...any), args []any) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "event: listener panic recovered: %v\n", r)
		}
	}()
	listener(args...)
}
