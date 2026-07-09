package broker

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/crypto"
	"github.com/dangduoc08/ginject/matcher"
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

type subscription struct {
	id         string
	topic      string
	pattern    matcher.Pattern
	handler    MessageHandler
	isOnce     bool
	fired      atomic.Bool
	queueGroup string
	broker     *MemoryBroker
}

func (s *subscription) ID() string         { return s.id }
func (s *subscription) Topic() string      { return s.topic }
func (s *subscription) Unsubscribe() error { return s.broker.Unsubscribe(s) }

type complexGroup struct {
	pattern  matcher.Pattern
	subsByID map[string]*subscription
}

type queueGroup struct {
	subs      []*subscription
	indexByID map[string]int
	counter   atomic.Uint64
}

func newQueueGroup() *queueGroup {
	g := &queueGroup{indexByID: make(map[string]int)}
	g.counter.Store(^uint64(0))
	return g
}

func (g *queueGroup) add(sub *subscription) {
	g.indexByID[sub.id] = len(g.subs)
	g.subs = append(g.subs, sub)
}

func (g *queueGroup) remove(id string) {
	idx, ok := g.indexByID[id]
	if !ok {
		return
	}
	last := len(g.subs) - 1
	if idx != last {
		g.subs[idx] = g.subs[last]
		g.indexByID[g.subs[idx].id] = idx
	}
	g.subs[last] = nil
	g.subs = g.subs[:last]
	delete(g.indexByID, id)
}

func (g *queueGroup) pick() *subscription {
	if len(g.subs) == 0 {
		return nil
	}
	return g.subs[g.counter.Add(1)%uint64(len(g.subs))]
}

type asyncJob struct {
	topic   string
	payload any
}

type MemoryBroker struct {
	mu      sync.RWMutex
	closeMu sync.RWMutex
	// exactByTopic holds exact-match subscriptions, keyed by topic; each inner map is keyed by subscription ID.
	exactByTopic map[string]map[string]*subscription
	// prefixByPrefix holds single-suffix-wildcard subscriptions, keyed by topic prefix; each inner map is keyed by subscription ID.
	prefixByPrefix map[string]map[string]*subscription
	globalByID     map[string]*subscription
	complexByTopic map[string]*complexGroup
	// queueGroupsByTopic holds queue groups keyed by topic; each inner map is keyed by group name.
	queueGroupsByTopic map[string]map[string]*queueGroup
	closed             atomic.Bool
	cfg                Config
	stats              brokerStats
	asyncCh            chan asyncJob
	wg                 sync.WaitGroup
}

func New() Broker {
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
			b.wg.Go(func() {
				for job := range b.asyncCh {
					_ = b.publishInternal(job.topic, job.payload)
				}
			})
		}
	}
	return b
}

func newID() string {
	id, err := crypto.UUID()
	if err != nil {
		panic(color.FmtRed("broker: failed to generate UUID: %v", err))
	}
	return id
}

func lastDot(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '.' {
			return i
		}
	}
	return -1
}

func (b *MemoryBroker) removeFromBucket(sub *subscription) {
	switch sub.pattern.Kind() {
	case matcher.KindGlobal:
		delete(b.globalByID, sub.id)
	case matcher.KindSingleSuffix:
		pfx := sub.pattern.SimplePrefix()
		delete(b.prefixByPrefix[pfx], sub.id)
		if len(b.prefixByPrefix[pfx]) == 0 {
			delete(b.prefixByPrefix, pfx)
		}
	case matcher.KindExact:
		delete(b.exactByTopic[sub.topic], sub.id)
		if len(b.exactByTopic[sub.topic]) == 0 {
			delete(b.exactByTopic, sub.topic)
		}
	case matcher.KindComplex:
		if cg := b.complexByTopic[sub.topic]; cg != nil {
			delete(cg.subsByID, sub.id)
			if len(cg.subsByID) == 0 {
				delete(b.complexByTopic, sub.topic)
			}
		}
	}
}

func (b *MemoryBroker) callHandler(h MessageHandler, msg *Message) {
	if !b.cfg.RecoverPanics {
		h(msg)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			b.runOnPanic(msg, r)
		}
	}()
	h(msg)
}

func (b *MemoryBroker) runOnPanic(msg *Message, r any) {
	if b.cfg.OnPanic == nil {
		return
	}
	defer func() { _ = recover() }()
	b.cfg.OnPanic(msg, r)
}

func (b *MemoryBroker) runBeforePublish(topic string, payload any) {
	defer func() { _ = recover() }()
	b.cfg.BeforePublish(topic, payload)
}

func (b *MemoryBroker) runAfterPublish(topic string, payload any, err error) {
	defer func() { _ = recover() }()
	b.cfg.AfterPublish(topic, payload, err)
}

func (b *MemoryBroker) runBeforeDispatch(msg *Message, i int) {
	defer func() { _ = recover() }()
	b.cfg.BeforeDispatch(msg, i)
}

func (b *MemoryBroker) runAfterDispatch(msg *Message, i int) {
	defer func() { _ = recover() }()
	b.cfg.AfterDispatch(msg, i)
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
	var handlers []MessageHandler
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
	if dot := lastDot(topic); dot >= 0 {
		for _, sub := range b.prefixByPrefix[topic[:dot]] {
			addSub(sub)
		}
	}
	for _, sub := range b.globalByID {
		addSub(sub)
	}
	for _, cg := range b.complexByTopic {
		if matcher.Match(cg.pattern, topic) {
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

	pat := matcher.Parse(topic)
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
	case matcher.KindGlobal:
		b.globalByID[sub.id] = sub
	case matcher.KindSingleSuffix:
		pfx := pat.SimplePrefix()
		if b.prefixByPrefix[pfx] == nil {
			b.prefixByPrefix[pfx] = make(map[string]*subscription)
		}
		b.prefixByPrefix[pfx][sub.id] = sub
	case matcher.KindExact:
		if b.exactByTopic[topic] == nil {
			b.exactByTopic[topic] = make(map[string]*subscription)
		}
		b.exactByTopic[topic][sub.id] = sub
	case matcher.KindComplex:
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
	if !matcher.Parse(topic).IsExact() {
		return nil, ErrWildcardInQueue
	}

	pat := matcher.Parse(topic)
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
	pat := matcher.Parse(topic)
	b.mu.Lock()
	switch pat.Kind() {
	case matcher.KindGlobal:
		b.globalByID = make(map[string]*subscription)
	case matcher.KindSingleSuffix:
		delete(b.prefixByPrefix, pat.SimplePrefix())
	case matcher.KindExact:
		delete(b.exactByTopic, topic)
		delete(b.queueGroupsByTopic, topic)
	case matcher.KindComplex:
		delete(b.complexByTopic, topic)
	}
	b.mu.Unlock()
	return nil
}

func (b *MemoryBroker) ListenerCount(topic string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	pat := matcher.Parse(topic)
	switch pat.Kind() {
	case matcher.KindGlobal:
		return len(b.globalByID)
	case matcher.KindSingleSuffix:
		return len(b.prefixByPrefix[pat.SimplePrefix()])
	case matcher.KindExact:
		n := len(b.exactByTopic[topic])
		for _, g := range b.queueGroupsByTopic[topic] {
			n += len(g.subs)
		}
		return n
	case matcher.KindComplex:
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
