package memorybroker

import (
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrClosed          = errors.New("memorybroker: broker is closed")
	ErrNilHandler      = errors.New("memorybroker: handler must not be nil")
	ErrEmptyTopic      = errors.New("memorybroker: topic must not be empty")
	ErrEmptyGroup      = errors.New("memorybroker: group must not be empty")
	ErrAsyncQueueFull  = errors.New("memorybroker: async queue full")
	ErrNoAsyncWorkers  = errors.New("memorybroker: PublishAsync requires AsyncWorkers > 0")
	ErrWildcardInQueue = errors.New("memorybroker: SubscribeQueue requires an exact topic")
)

type Message struct {
	ID        string
	Topic     string
	Payload   any
	Timestamp time.Time
	Metadata  map[string]any
}

type MessageHandler func(*Message)

type Subscription interface {
	ID() string
	Topic() string
	Unsubscribe() error
}

type brokerStats struct {
	messagesSent    atomic.Uint64
	messagesDropped atomic.Uint64
	publishCalls    atomic.Uint64
}

type Stats struct {
	Topics          int
	Subscribers     int
	MessagesSent    uint64
	MessagesDropped uint64
	PublishCalls    uint64
}

type Broker interface {
	Publish(topic string, payload any) error
	PublishAsync(topic string, payload any) error
	Subscribe(topic string, handler MessageHandler) (Subscription, error)
	Once(topic string, handler MessageHandler) (Subscription, error)
	SubscribeQueue(topic, group string, handler MessageHandler) (Subscription, error)
	Unsubscribe(sub Subscription) error
	Off(topic string) error
	ListenerCount(topic string) int
	Topics() []string
	Clear() error
	Close() error
	Stats() Stats
}
