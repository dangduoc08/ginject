package memorybroker

import (
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrClosed          = errors.New("broker: broker is closed")
	ErrNilHandler      = errors.New("broker: handler must not be nil")
	ErrEmptyTopic      = errors.New("broker: topic must not be empty")
	ErrEmptyGroup      = errors.New("broker: group must not be empty")
	ErrAsyncQueueFull  = errors.New("broker: async queue full")
	ErrNoAsyncWorkers  = errors.New("broker: PublishAsync requires AsyncWorkers > 0")
	ErrWildcardInQueue = errors.New("broker: SubscribeQueue requires an exact topic")
)

type Config struct {
	RecoverPanics  bool
	OnPanic        func(*Message, any)
	AsyncWorkers   int
	AsyncQueueSize int
	BeforePublish  func(topic string, payload any)
	AfterPublish   func(topic string, payload any, err error)
	BeforeDispatch func(msg *Message, handler int)
	AfterDispatch  func(msg *Message, handler int)
}

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
