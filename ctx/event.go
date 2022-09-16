package ctx

const (
	REQUEST_FINISHED = "REQUEST_FINISHED"
)

type event struct {
	opts     map[string]func(args ...interface{})
	onceOpts map[string]func(args ...interface{})
}

func newEvent() *event {
	return &event{
		opts:     make(map[string]func(args ...interface{})),
		onceOpts: make(map[string]func(args ...interface{})),
	}
}

func (e *event) On(eventName string, listener func(args ...interface{})) {
	e.opts[eventName] = listener
}

func (e *event) Once(eventName string, listener func(args ...interface{})) {
	e.onceOpts[eventName] = listener
}

func (e *event) Emit(eventName string, args ...interface{}) {
	ch1 := make(chan bool)
	ch2 := make(chan bool)
	defer close(ch1)
	defer close(ch2)

	go (func(c chan bool) {
		if e.opts[eventName] != nil {
			listener := e.opts[eventName]
			listener(args...)
		}
		c <- true
	})(ch1)

	go (func(c chan bool) {
		if e.onceOpts[eventName] != nil {
			listener := e.onceOpts[eventName]
			listener(args...)
			delete(e.onceOpts, eventName)
		}
		c <- true
	})(ch2)

	<-ch1
	<-ch2
}
