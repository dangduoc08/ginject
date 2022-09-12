package ctx

type Event struct {
	options map[string]func(args ...interface{})
}

func NewEvent() *Event {
	return &Event{
		options: make(map[string]func(args ...interface{})),
	}
}

func (event *Event) On(eventName string, handler func(args ...interface{})) {
	event.options[eventName] = handler
}

func (event *Event) Emit(eventName string, args ...interface{}) {
	if event.options[eventName] != nil {
		handler := event.options[eventName]
		handler(args...)
	}
}
