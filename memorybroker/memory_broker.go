package memorybroker

import (
	"runtime"
	"sync"
	"sync/atomic"

	ptrn "github.com/dangduoc08/ginject/pattern"
)

type asyncJob struct {
	topic   string
	payload any
}

type MemoryBroker struct {
	mu                 sync.RWMutex
	closeMu            sync.RWMutex
	exactByTopic       map[string]map[string]*subscription
	prefixByPrefix     map[string]map[string]*subscription
	globalByID         map[string]*subscription
	complexByTopic     map[string]*complexGroup
	queueGroupsByTopic map[string]map[string]*queueGroup
	closed             atomic.Bool
	cfg                Config
	stats              brokerStats
	asyncCh            chan asyncJob
	wg                 sync.WaitGroup
}

func NewMemoryBroker() Broker {
	workers := runtime.GOMAXPROCS(0)
	return NewWithConfig(Config{
		RecoverPanics:  true,
		AsyncWorkers:   workers,
		AsyncQueueSize: workers * 64,
	})
}

func NewWithConfig(cfg Config) Broker {
	b := &MemoryBroker{
		exactByTopic:       make(map[string]map[string]*subscription),
		prefixByPrefix:     make(map[string]map[string]*subscription),
		globalByID:         make(map[string]*subscription),
		complexByTopic:     make(map[string]*complexGroup),
		queueGroupsByTopic: make(map[string]map[string]*queueGroup),
		cfg:                cfg,
	}
	if cfg.AsyncWorkers > 0 {
		qSize := cfg.AsyncQueueSize
		if qSize <= 0 {
			qSize = cfg.AsyncWorkers * 64
		}
		b.asyncCh = make(chan asyncJob, qSize)
		for range cfg.AsyncWorkers {
			b.wg.Add(1)
			go func() {
				defer b.wg.Done()
				for job := range b.asyncCh {
					_ = b.publishInternal(job.topic, job.payload)
				}
			}()
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
	if b.asyncCh == nil {
		return ErrNoAsyncWorkers
	}
	b.closeMu.RLock()
	if b.closed.Load() {
		b.closeMu.RUnlock()
		return ErrClosed
	}
	var err error
	select {
	case b.asyncCh <- asyncJob{topic, payload}:
	default:
		b.stats.messagesDropped.Add(1)
		err = ErrAsyncQueueFull
	}
	b.closeMu.RUnlock()
	return err
}

func (b *MemoryBroker) subscribe(topic string, handler MessageHandler, once bool) (Subscription, error) {
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
		isOnce:  once,
		broker:  b,
	}

	b.mu.Lock()
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
	b.mu.Unlock()

	return sub, nil
}

func (b *MemoryBroker) SubscribeQueue(topic, group string, handler MessageHandler) (Subscription, error) {
	if topic == "" {
		return nil, ErrEmptyTopic
	}
	if group == "" {
		return nil, ErrEmptyGroup
	}
	if handler == nil {
		return nil, ErrNilHandler
	}
	if b.closed.Load() {
		return nil, ErrClosed
	}
	if !ptrn.NewPattern(topic).IsExact() {
		return nil, ErrWildcardInQueue
	}

	pat := ptrn.NewPattern(topic)
	sub := &subscription{
		id:         newID(),
		topic:      topic,
		pattern:    pat,
		handler:    handler,
		queueGroup: group,
		broker:     b,
	}

	b.mu.Lock()
	if b.queueGroupsByTopic[topic] == nil {
		b.queueGroupsByTopic[topic] = make(map[string]*queueGroup)
	}
	if b.queueGroupsByTopic[topic][group] == nil {
		b.queueGroupsByTopic[topic][group] = newQueueGroup()
	}
	b.queueGroupsByTopic[topic][group].add(sub)
	b.mu.Unlock()

	return sub, nil
}

func (b *MemoryBroker) Subscribe(topic string, handler MessageHandler) (Subscription, error) {
	return b.subscribe(topic, handler, false)
}

func (b *MemoryBroker) Once(topic string, handler MessageHandler) (Subscription, error) {
	return b.subscribe(topic, handler, true)
}

func (b *MemoryBroker) Unsubscribe(sub Subscription) error {
	if b.closed.Load() {
		return ErrClosed
	}
	if sub == nil {
		return nil
	}
	s, ok := sub.(*subscription)
	if !ok {
		return nil
	}

	b.mu.Lock()
	if s.queueGroup != "" {
		if groups, ok := b.queueGroupsByTopic[s.topic]; ok {
			if g := groups[s.queueGroup]; g != nil {
				g.remove(s.id)
				if len(g.subs) == 0 {
					delete(groups, s.queueGroup)
					if len(groups) == 0 {
						delete(b.queueGroupsByTopic, s.topic)
					}
				}
			}
		}
	} else {
		b.removeFromBucket(s)
	}
	b.mu.Unlock()

	return nil
}

func (b *MemoryBroker) Off(topic string) error {
	if b.closed.Load() {
		return ErrClosed
	}
	pat := ptrn.NewPattern(topic)
	b.mu.Lock()
	switch pat.Kind() {
	case ptrn.KindGlobal:
		b.globalByID = make(map[string]*subscription)
	case ptrn.KindSuffixWildcard:
		delete(b.prefixByPrefix, pat.SimplePrefix())
	case ptrn.KindExact:
		delete(b.exactByTopic, topic)
		delete(b.queueGroupsByTopic, topic)
	case ptrn.KindComplex:
		delete(b.complexByTopic, topic)
	}
	b.mu.Unlock()
	return nil
}

func (b *MemoryBroker) ListenerCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	pat := ptrn.NewPattern(topic)
	switch pat.Kind() {
	case ptrn.KindGlobal:
		return len(b.globalByID)
	case ptrn.KindSuffixWildcard:
		return len(b.prefixByPrefix[pat.SimplePrefix()])
	case ptrn.KindExact:
		n := len(b.exactByTopic[topic])
		for _, g := range b.queueGroupsByTopic[topic] {
			n += len(g.subs)
		}
		return n
	case ptrn.KindComplex:
		if cg := b.complexByTopic[topic]; cg != nil {
			return len(cg.subsByID)
		}
		return 0
	}
	return 0
}

func (b *MemoryBroker) Topics() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	seen := make(map[string]struct{})
	var topics []string

	add := func(t string) {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			topics = append(topics, t)
		}
	}

	for t, m := range b.exactByTopic {
		if len(m) > 0 {
			add(t)
		}
	}
	for t, groups := range b.queueGroupsByTopic {
		for _, g := range groups {
			if len(g.subs) > 0 {
				add(t)
				break
			}
		}
	}
	for pfx, m := range b.prefixByPrefix {
		if len(m) > 0 {
			add(pfx + ".*")
		}
	}
	if len(b.globalByID) > 0 {
		add("*")
	}
	for rawPat, cg := range b.complexByTopic {
		if len(cg.subsByID) > 0 {
			add(rawPat)
		}
	}
	return topics
}

func (b *MemoryBroker) Clear() error {
	if b.closed.Load() {
		return ErrClosed
	}
	b.mu.Lock()
	b.exactByTopic = make(map[string]map[string]*subscription)
	b.prefixByPrefix = make(map[string]map[string]*subscription)
	b.globalByID = make(map[string]*subscription)
	b.complexByTopic = make(map[string]*complexGroup)
	b.queueGroupsByTopic = make(map[string]map[string]*queueGroup)
	b.mu.Unlock()
	return nil
}

func (b *MemoryBroker) Close() error {
	b.closeMu.Lock()
	b.closed.Store(true)
	if b.asyncCh != nil {
		close(b.asyncCh)
	}
	b.closeMu.Unlock()
	if b.asyncCh != nil {
		b.wg.Wait()
	}
	b.mu.Lock()
	b.exactByTopic = make(map[string]map[string]*subscription)
	b.prefixByPrefix = make(map[string]map[string]*subscription)
	b.globalByID = make(map[string]*subscription)
	b.complexByTopic = make(map[string]*complexGroup)
	b.queueGroupsByTopic = make(map[string]map[string]*queueGroup)
	b.mu.Unlock()
	return nil
}

func (b *MemoryBroker) Stats() Stats {
	b.mu.RLock()
	seenTopics := make(map[string]struct{})
	subs := 0

	for t, m := range b.exactByTopic {
		if len(m) > 0 {
			seenTopics[t] = struct{}{}
			subs += len(m)
		}
	}
	for t, groups := range b.queueGroupsByTopic {
		for _, g := range groups {
			if len(g.subs) > 0 {
				seenTopics[t] = struct{}{}
				subs += len(g.subs)
			}
		}
	}
	for pfx, m := range b.prefixByPrefix {
		if len(m) > 0 {
			seenTopics[pfx+".*"] = struct{}{}
			subs += len(m)
		}
	}
	if len(b.globalByID) > 0 {
		seenTopics["*"] = struct{}{}
		subs += len(b.globalByID)
	}
	for rawPat, cg := range b.complexByTopic {
		if len(cg.subsByID) > 0 {
			seenTopics[rawPat] = struct{}{}
			subs += len(cg.subsByID)
		}
	}
	b.mu.RUnlock()

	return Stats{
		Topics:          len(seenTopics),
		Subscribers:     subs,
		MessagesSent:    b.stats.messagesSent.Load(),
		MessagesDropped: b.stats.messagesDropped.Load(),
		PublishCalls:    b.stats.publishCalls.Load(),
	}
}
