package memorybroker

import (
	"sync"
	"sync/atomic"
	"time"

	ptrn "github.com/dangduoc08/ginject/pattern"
)

type MemoryBroker struct {
	rwMu           sync.RWMutex
	exactByTopic   map[string]map[string]*subscription
	prefixByPrefix map[string]map[string]*subscription
	globalByID     map[string]*subscription
	complexByTopic map[string]*complexGroup
	closed         atomic.Bool
	wg             sync.WaitGroup
}

func NewMemoryBroker() Broker {
	return &MemoryBroker{
		exactByTopic:   make(map[string]map[string]*subscription),
		prefixByPrefix: make(map[string]map[string]*subscription),
		globalByID:     make(map[string]*subscription),
		complexByTopic: make(map[string]*complexGroup),
	}
}

func (b *MemoryBroker) Publish(topic string, payload any) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	if b.closed.Load() {
		return ErrClosed
	}
	return b.publishInternal(topic, payload)
}

func (b *MemoryBroker) PublishAsync(topic string, payload any) error {
	if topic == "" {
		return ErrEmptyTopic
	}

	b.rwMu.RLock()
	if b.closed.Load() {
		b.rwMu.RUnlock()
		return ErrClosed
	}
	b.wg.Add(1)
	b.rwMu.RUnlock()

	go func() {
		defer b.wg.Done()
		_ = b.publishInternal(topic, payload)
	}()
	return nil
}

func (b *MemoryBroker) Subscribe(topic string, handler MessageHandler) (Subscription, error) {
	if topic == "" {
		return nil, ErrEmptyTopic
	}
	if handler == nil {
		return nil, ErrNilHandler
	}
	if b.closed.Load() {
		return nil, ErrClosed
	}

	pat := ptrn.NewPattern(topic)
	sub := &subscription{
		id:      newID(),
		topic:   topic,
		pattern: pat,
		handler: handler,
		broker:  b,
	}

	b.rwMu.Lock()
	if b.closed.Load() {
		b.rwMu.Unlock()
		return nil, ErrClosed
	}
	switch pat.Kind() {
	case ptrn.KindGlobal:
		b.globalByID[sub.id] = sub
	case ptrn.KindSuffixWildcard:
		pfx := pat.SimplePrefix()
		if b.prefixByPrefix[pfx] == nil {
			b.prefixByPrefix[pfx] = make(map[string]*subscription)
		}
		b.prefixByPrefix[pfx][sub.id] = sub
	case ptrn.KindExact:
		if b.exactByTopic[topic] == nil {
			b.exactByTopic[topic] = make(map[string]*subscription)
		}
		b.exactByTopic[topic][sub.id] = sub
	case ptrn.KindComplex:
		if b.complexByTopic[topic] == nil {
			b.complexByTopic[topic] = &complexGroup{pattern: pat, subsByID: make(map[string]*subscription)}
		}
		b.complexByTopic[topic].subsByID[sub.id] = sub
	}
	b.rwMu.Unlock()

	return sub, nil
}

func (b *MemoryBroker) Unsubscribe(sub Subscription) error {
	if sub == nil {
		return nil
	}
	s, ok := sub.(*subscription)
	if !ok || s.broker != b {
		if b.closed.Load() {
			return ErrClosed
		}
		if !ok {
			return nil
		}
		return ErrForeignSubscription
	}

	b.rwMu.Lock()
	defer b.rwMu.Unlock()

	if b.closed.Load() {
		return ErrClosed
	}
	b.removeFromBucket(s)
	return nil
}

func (b *MemoryBroker) Close() error {
	b.rwMu.Lock()
	wasOpen := b.closed.CompareAndSwap(false, true)
	b.rwMu.Unlock()
	if !wasOpen {
		return nil
	}

	b.wg.Wait()

	b.rwMu.Lock()
	b.exactByTopic = make(map[string]map[string]*subscription)
	b.prefixByPrefix = make(map[string]map[string]*subscription)
	b.globalByID = make(map[string]*subscription)
	b.complexByTopic = make(map[string]*complexGroup)
	b.rwMu.Unlock()
	return nil
}

func (b *MemoryBroker) callHandler(h MessageHandler, msg *Message) {
	defer func() { _ = recover() }()
	h(msg)
}

func (b *MemoryBroker) publishInternal(topic string, payload any) error {
	now := time.Now()

	b.rwMu.RLock()
	total := len(b.exactByTopic[topic]) + len(b.globalByID)
	forEachPrefixOf(topic, func(prefix string) {
		total += len(b.prefixByPrefix[prefix])
	})
	var matchedComplex []*complexGroup
	for _, cg := range b.complexByTopic {
		if cg.pattern.Match(topic) {
			matchedComplex = append(matchedComplex, cg)
			total += len(cg.subsByID)
		}
	}

	if total == 0 {
		b.rwMu.RUnlock()
		return nil
	}

	handlers := make([]MessageHandler, 0, total)
	for _, sub := range b.exactByTopic[topic] {
		handlers = append(handlers, sub.handler)
	}
	forEachPrefixOf(topic, func(prefix string) {
		for _, sub := range b.prefixByPrefix[prefix] {
			handlers = append(handlers, sub.handler)
		}
	})
	for _, sub := range b.globalByID {
		handlers = append(handlers, sub.handler)
	}
	for _, cg := range matchedComplex {
		for _, sub := range cg.subsByID {
			handlers = append(handlers, sub.handler)
		}
	}
	b.rwMu.RUnlock()

	msg := &Message{
		Topic:     topic,
		Payload:   payload,
		Timestamp: now,
	}
	for _, h := range handlers {
		b.callHandler(h, msg)
	}

	return nil
}

func (b *MemoryBroker) removeFromBucket(sub *subscription) {
	switch sub.pattern.Kind() {
	case ptrn.KindGlobal:
		delete(b.globalByID, sub.id)
	case ptrn.KindSuffixWildcard:
		pfx := sub.pattern.SimplePrefix()
		delete(b.prefixByPrefix[pfx], sub.id)
		if len(b.prefixByPrefix[pfx]) == 0 {
			delete(b.prefixByPrefix, pfx)
		}
	case ptrn.KindExact:
		delete(b.exactByTopic[sub.topic], sub.id)
		if len(b.exactByTopic[sub.topic]) == 0 {
			delete(b.exactByTopic, sub.topic)
		}
	case ptrn.KindComplex:
		if cg := b.complexByTopic[sub.topic]; cg != nil {
			delete(cg.subsByID, sub.id)
			if len(cg.subsByID) == 0 {
				delete(b.complexByTopic, sub.topic)
			}
		}
	}
}
