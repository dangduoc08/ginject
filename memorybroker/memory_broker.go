package memorybroker

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dangduoc08/ginject/internal/color"
	ptrn "github.com/dangduoc08/ginject/pattern"
)

type PanicHandler func(topic string, recovered any)

type Option func(*MemoryBroker)

// WithPanicHandler replaces the default stderr report for panics escaping a
// subscriber. Handlers stay isolated either way.
func WithPanicHandler(fn PanicHandler) Option {
	return func(b *MemoryBroker) {
		b.onPanic = fn
	}
}

type MemoryBroker struct {
	rwMu           sync.RWMutex
	exactByTopic   map[string]map[string]*subscription
	prefixByPrefix map[string]map[string]*subscription
	globalByID     map[string]*subscription
	complexByTopic map[string]*complexGroup
	onPanic        PanicHandler
	nPrefixSubs    int
	closed         atomic.Bool
	wg             sync.WaitGroup
}

func NewMemoryBroker(opts ...Option) Broker {
	b := &MemoryBroker{
		exactByTopic:   make(map[string]map[string]*subscription),
		prefixByPrefix: make(map[string]map[string]*subscription),
		globalByID:     make(map[string]*subscription),
		complexByTopic: make(map[string]*complexGroup),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
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
		bucket := b.prefixByPrefix[pfx]
		if bucket == nil {
			bucket = make(map[string]*subscription)
			b.prefixByPrefix[pfx] = bucket
		}
		bucket[sub.id] = sub
		b.nPrefixSubs++
	case ptrn.KindExact:
		bucket := b.exactByTopic[topic]
		if bucket == nil {
			bucket = make(map[string]*subscription)
			b.exactByTopic[topic] = bucket
		}
		bucket[sub.id] = sub
	case ptrn.KindComplex:
		cg := b.complexByTopic[topic]
		if cg == nil {
			cg = &complexGroup{pattern: pat, subsByID: make(map[string]*subscription)}
			b.complexByTopic[topic] = cg
		}
		cg.subsByID[sub.id] = sub
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
	b.nPrefixSubs = 0
	b.rwMu.Unlock()
	return nil
}

func (b *MemoryBroker) callHandler(h MessageHandler, msg *Message) {
	defer func() {
		if rec := recover(); rec != nil {
			b.reportPanic(msg.Topic, rec)
		}
	}()
	h(msg)
}

func (b *MemoryBroker) reportPanic(topic string, rec any) {
	defer func() { _ = recover() }()

	if b.onPanic != nil {
		b.onPanic(topic, rec)
		return
	}

	fmt.Fprintln(os.Stderr, color.FmtRed("memorybroker: subscriber panic recovered on topic '%v': %v", topic, rec))
}

func (b *MemoryBroker) publishInternal(topic string, payload any) error {
	b.rwMu.RLock()

	exact := b.exactByTopic[topic]
	nDirect := len(exact) + len(b.globalByID)
	hasPrefix := len(b.prefixByPrefix) > 0
	hasComplex := len(b.complexByTopic) > 0

	if nDirect == 0 && !hasPrefix && !hasComplex {
		b.rwMu.RUnlock()
		return nil
	}

	var handlers []MessageHandler
	if total := nDirect + b.nPrefixSubs; total > 0 {
		handlers = make([]MessageHandler, 0, total)
	}
	for _, sub := range exact {
		handlers = append(handlers, sub.handler)
	}
	for _, sub := range b.globalByID {
		handlers = append(handlers, sub.handler)
	}
	if hasPrefix {
		for i := strings.LastIndexByte(topic, '.'); i >= 0; i = strings.LastIndexByte(topic[:i], '.') {
			for _, sub := range b.prefixByPrefix[topic[:i]] {
				handlers = append(handlers, sub.handler)
			}
		}
	}
	if hasComplex {
		for _, cg := range b.complexByTopic {
			if cg.pattern.Match(topic) {
				for _, sub := range cg.subsByID {
					handlers = append(handlers, sub.handler)
				}
			}
		}
	}
	b.rwMu.RUnlock()

	if len(handlers) == 0 {
		return nil
	}

	msg := &Message{
		Topic:     topic,
		Payload:   payload,
		Timestamp: time.Now(),
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
		bucket := b.prefixByPrefix[pfx]
		if _, ok := bucket[sub.id]; ok {
			delete(bucket, sub.id)
			b.nPrefixSubs--
		}
		if len(bucket) == 0 {
			delete(b.prefixByPrefix, pfx)
		}
	case ptrn.KindExact:
		bucket := b.exactByTopic[sub.topic]
		delete(bucket, sub.id)
		if len(bucket) == 0 {
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
