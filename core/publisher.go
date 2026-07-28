package core

import "github.com/dangduoc08/ginject/broker"

type publisher struct {
	broker broker.Broker
}

func newPublisher(br broker.Broker) *publisher {
	return &publisher{broker: br}
}

func (p *publisher) Publish(topic string, payload ...any) error {
	var pl any
	if len(payload) > 0 {
		pl = payload[0]
	}
	return p.broker.Publish(topic, pl)
}
