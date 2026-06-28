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
	GetSubscriptions() map[string][]uint64
}

type builtInBroker struct {
	subscription *Subscription
}

func NewBroker() Broker {
	return &builtInBroker{
		subscription: NewSubscription(),
	}
}

func (b *builtInBroker) Subscribe(topic string, handler MessageHandler) (uint64, error) {
	if topic == "" {
		return 0, ErrEmptyTopic
	}
	if handler == nil {
		return 0, ErrNilHandler
	}

	return b.subscription.insert(topic, handler), nil
}

func (b *builtInBroker) Publish(topic string, payload any) error {
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

func (b *builtInBroker) PublishAsync(topic string, payload any) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	go func() {
		_ = b.Publish(topic, payload)
	}()

	return nil
}

func (b *builtInBroker) Unsubscribe(topic string, id uint64) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	b.subscription.remove(topic, id)
	return nil
}

func (b *builtInBroker) GetSubscriptions() map[string][]uint64 {
	return b.subscription.list()
}
