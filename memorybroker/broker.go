package memorybroker

import (
	"errors"
	"time"
)

var (
	ErrClosed              = errors.New("memorybroker: broker is closed")
	ErrNilHandler          = errors.New("memorybroker: handler must not be nil")
	ErrEmptyTopic          = errors.New("memorybroker: topic must not be empty")
	ErrForeignSubscription = errors.New("memorybroker: subscription does not belong to this broker")
)

type Message struct {
	Topic     string
	Payload   any
	Timestamp time.Time
}

type MessageHandler func(*Message)

type Broker interface {
	Subscribe(topic string, handler MessageHandler) (Subscription, error)
	Unsubscribe(sub Subscription) error
	Publish(topic string, payload any) error
	PublishAsync(topic string, payload any) error
	Close() error
}
