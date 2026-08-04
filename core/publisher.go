package core

import "github.com/dangduoc08/ginject/memorybroker"

type publisher struct {
	broker memorybroker.Broker
}

func newPublisher(br memorybroker.Broker) *publisher {
	return &publisher{broker: br}
}

func (p *publisher) Publish(topic string, payload ...any) error {
	var pl any
	if len(payload) > 0 {
		pl = payload[0]
	}
	return p.broker.Publish(topic, pl)
}
