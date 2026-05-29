package ctx

import (
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/dangduoc08/ginject/utils"
)

const (
	REQUEST_FINISHED    = "REQUEST_FINISHED"
	CATCH_EXCEPTION     = "CATCH_EXCEPTION"
	defaultMaxListeners = 10
)

type event struct {
	mu           sync.RWMutex
	opts         map[string][]func(args ...any)
	onceOpts     map[string][]func(args ...any)
	maxListeners int
}

func NewEvent() *event {
	return &event{
		opts:         make(map[string][]func(args ...any)),
		onceOpts:     make(map[string][]func(args ...any)),
		maxListeners: defaultMaxListeners,
	}
}

func (e *event) reset() {
	e.mu.Lock()
	clear(e.opts)
	clear(e.onceOpts)
	e.mu.Unlock()
}

func (e *event) SetMaxListeners(n int) {
	e.mu.Lock()
	e.maxListeners = n
	e.mu.Unlock()
}

func (e *event) On(eventName string, fn func(args ...any)) {
	e.mu.Lock()
	e.opts[eventName] = append(e.opts[eventName], fn)
	n := len(e.opts[eventName]) + len(e.onceOpts[eventName])
	max := e.maxListeners
	e.mu.Unlock()
	if max > 0 && n > max {
		fmt.Fprintln(os.Stderr, utils.FmtYellow(
			"warning: possible EventEmitter memory leak detected. %d '%v' listeners added. Use SetMaxListeners to increase limit",
			n, eventName,
		))
	}
}

func (e *event) Once(eventName string, fn func(args ...any)) {
	e.mu.Lock()
	e.onceOpts[eventName] = append(e.onceOpts[eventName], fn)
	n := len(e.opts[eventName]) + len(e.onceOpts[eventName])
	max := e.maxListeners
	e.mu.Unlock()
	if max > 0 && n > max {
		fmt.Fprintln(os.Stderr, utils.FmtYellow(
			"warning: possible EventEmitter memory leak detected. %d '%v' listeners added. Use SetMaxListeners to increase limit",
			n, eventName,
		))
	}
}

func (e *event) Off(eventName string, fn func(args ...any)) {
	ptr := reflect.ValueOf(fn).Pointer()
	e.mu.Lock()
	defer e.mu.Unlock()

	if listeners, ok := e.opts[eventName]; ok {
		for i, l := range listeners {
			if reflect.ValueOf(l).Pointer() == ptr {
				e.opts[eventName] = append(listeners[:i], listeners[i+1:]...)
				return
			}
		}
	}
	if onceListeners, ok := e.onceOpts[eventName]; ok {
		for i, l := range onceListeners {
			if reflect.ValueOf(l).Pointer() == ptr {
				e.onceOpts[eventName] = append(onceListeners[:i], onceListeners[i+1:]...)
				return
			}
		}
	}
}

func (e *event) RemoveAllListeners(eventName string) {
	e.mu.Lock()
	delete(e.opts, eventName)
	delete(e.onceOpts, eventName)
	e.mu.Unlock()
}

func (e *event) Emit(eventName string, args ...any) {
	e.mu.Lock()

	src := e.opts[eventName]
	var listeners []func(args ...any)
	if len(src) > 0 {
		listeners = make([]func(args ...any), len(src))
		copy(listeners, src)
	}

	onceListeners := e.onceOpts[eventName]
	if len(onceListeners) > 0 {
		delete(e.onceOpts, eventName)
	}

	e.mu.Unlock()

	for _, l := range listeners {
		callSafe(l, args)
	}
	for _, l := range onceListeners {
		callSafe(l, args)
	}
}

func (e *event) ListenerCount(eventName string) int {
	e.mu.RLock()
	n := len(e.opts[eventName]) + len(e.onceOpts[eventName])
	e.mu.RUnlock()
	return n
}

func (e *event) HasListeners(eventName string) bool {
	e.mu.RLock()
	has := len(e.opts[eventName]) > 0 || len(e.onceOpts[eventName]) > 0
	e.mu.RUnlock()
	return has
}

func (e *event) EventNames() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	seen := make(map[string]struct{}, len(e.opts)+len(e.onceOpts))
	for k, v := range e.opts {
		if len(v) > 0 {
			seen[k] = struct{}{}
		}
	}
	for k, v := range e.onceOpts {
		if len(v) > 0 {
			seen[k] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for k := range seen {
		names = append(names, k)
	}
	return names
}

func callSafe(l func(...any), args []any) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, utils.FmtRed("EventEmitter: listener panic recovered: %v", r))
		}
	}()
	l(args...)
}
