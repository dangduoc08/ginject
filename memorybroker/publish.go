package memorybroker

import "time"

func (b *MemoryBroker) publishInternal(topic string, payload any) error {
	b.stats.publishCalls.Add(1)

	if b.cfg.BeforePublish != nil {
		b.runBeforePublish(topic, payload)
	}

	msg := &Message{
		ID:        newID(),
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
	}

	b.mu.RLock()
	cap := len(b.exactByTopic[topic]) + len(b.globalByID)
	if cap < 16 {
		cap = 16
	}
	handlers := make([]MessageHandler, 0, cap)
	var onceSubs []*subscription

	addSub := func(sub *subscription) {
		if !sub.isOnce {
			handlers = append(handlers, sub.handler)
			return
		}
		if sub.fired.CompareAndSwap(false, true) {
			handlers = append(handlers, sub.handler)
			onceSubs = append(onceSubs, sub)
		}
	}

	for _, sub := range b.exactByTopic[topic] {
		addSub(sub)
	}
	forEachPrefixOf(topic, func(prefix string) {
		for _, sub := range b.prefixByPrefix[prefix] {
			addSub(sub)
		}
	})
	for _, sub := range b.globalByID {
		addSub(sub)
	}
	for _, cg := range b.complexByTopic {
		if cg.pattern.Match(topic) {
			for _, sub := range cg.subsByID {
				addSub(sub)
			}
		}
	}
	for _, groups := range b.queueGroupsByTopic[topic] {
		if sub := groups.pick(); sub != nil {
			handlers = append(handlers, sub.handler)
		}
	}
	b.mu.RUnlock()

	b.stats.messagesSent.Add(uint64(len(handlers)))

	for i, h := range handlers {
		if b.cfg.BeforeDispatch != nil {
			b.runBeforeDispatch(msg, i)
		}
		b.callHandler(h, msg)
		if b.cfg.AfterDispatch != nil {
			b.runAfterDispatch(msg, i)
		}
	}

	if len(onceSubs) > 0 {
		b.mu.Lock()
		for _, sub := range onceSubs {
			b.removeFromBucket(sub)
		}
		b.mu.Unlock()
	}

	if b.cfg.AfterPublish != nil {
		b.runAfterPublish(topic, payload, nil)
	}

	return nil
}
