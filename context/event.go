package context

import (
	"sync"
)

const (
	REQUEST_FINISHED = "REQUEST_FINISHED"
	REQUEST_FAILED   = "REQUEST_FAILED"
)

type event struct {
	opts     *sync.Map
	onceOpts *sync.Map
}

func NewEvent() *event {
	return &event{
		opts:     &sync.Map{},
		onceOpts: &sync.Map{},
	}
}

func (e *event) On(eventName string, listener func(args ...interface{})) {
	e.opts.Store(eventName, listener)
}

func (e *event) Once(eventName string, listener func(args ...interface{})) {
	e.onceOpts.Store(eventName, listener)
}

func (e *event) Emit(eventName string, args ...interface{}) {
	ch := make(chan bool, 2)

	go (func(ch chan<- bool) {
		listener, ok := e.opts.Load(eventName)
		if ok {
			fn := listener.(func(args ...interface{}))
			fn(args...)
		}
		ch <- true
	})(ch)
	<-ch

	go (func(c chan<- bool) {
		listener, ok := e.onceOpts.Load(eventName)
		if ok {
			fn := listener.(func(args ...interface{}))
			fn(args...)
			e.onceOpts.Delete(eventName)
		}
		ch <- true
	})(ch)
	<-ch

	close(ch)
}
