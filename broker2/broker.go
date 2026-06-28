package broker2

import (
	"errors"
)

var (
	ErrEmptyTopic = errors.New("broker: topic must not be empty")
	ErrNilHandler = errors.New("broker: handler must not be nil")
)

type Broker interface {
	Subscribe(topic string, handler MessageHandler) (uint64, error)
	Publish(topic string, payload any) error
	PublishAsync(topic string, payload any) error
	Unsubscribe(topic string, id uint64) error
	Subscriptions() map[string][]uint64
}

type MemoryBroker struct {
	subscription *Subscription
}

func New() Broker {
	return &MemoryBroker{
		subscription: NewSubscription(),
	}
}

func (b *MemoryBroker) Subscribe(topic string, handler MessageHandler) (uint64, error) {
	if topic == "" {
		return 0, ErrEmptyTopic
	}
	if handler == nil {
		return 0, ErrNilHandler
	}

	return b.subscription.insert(topic, handler), nil
}

func (b *MemoryBroker) Publish(topic string, payload any) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	handlers := b.subscription.find(topic)
	if len(handlers) == 0 {
		return nil
	}

	msg := &Message{
		Topic:   topic,
		Payload: payload,
	}

	for _, h := range handlers {
		callHandler(h, msg)
	}

	return nil
}

func (b *MemoryBroker) PublishAsync(topic string, payload any) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	go func() {
		_ = b.Publish(topic, payload)
	}()

	return nil
}

func (b *MemoryBroker) Unsubscribe(topic string, id uint64) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	b.subscription.remove(topic, id)
	return nil
}

func (b *MemoryBroker) Subscriptions() map[string][]uint64 {
	return b.subscription.list()
}
