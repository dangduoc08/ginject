package ctx

import (
	"sync"
)

const (
	REQUEST_FINISHED = "REQUEST_FINISHED"
	CATCH_EXCEPTION  = "CATCH_EXCEPTION"
)

type event struct {
	mu       sync.RWMutex
	opts     map[string]func(args ...any)
	onceOpts map[string]func(args ...any)
}

func NewEvent() *event {
	return &event{
		opts:     make(map[string]func(args ...any)),
		onceOpts: make(map[string]func(args ...any)),
	}
}

func (e *event) reset() {
	e.mu.Lock()
	clear(e.opts)
	clear(e.onceOpts)
	e.mu.Unlock()
}

func (e *event) On(eventName string, listener func(args ...interface{})) {
	e.mu.Lock()
	e.opts[eventName] = listener
	e.mu.Unlock()
}

func (e *event) Once(eventName string, listener func(args ...interface{})) {
	e.mu.Lock()
	e.onceOpts[eventName] = listener
	e.mu.Unlock()
}

func (e *event) RemoveAllListeners(eventName string) {
	e.mu.Lock()
	delete(e.opts, eventName)
	delete(e.onceOpts, eventName)
	e.mu.Unlock()
}

func (e *event) Emit(eventName string, args ...interface{}) {
	e.mu.RLock()
	listener := e.opts[eventName]
	onceListener := e.onceOpts[eventName]
	e.mu.RUnlock()

	if listener != nil {
		listener(args...)
	}
	if onceListener != nil {
		onceListener(args...)
		e.mu.Lock()
		delete(e.onceOpts, eventName)
		e.mu.Unlock()
	}
}
