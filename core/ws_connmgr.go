package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/memorybroker"
	"golang.org/x/net/websocket"
)

const (
	DefaultWSSendBufferSize          = 32
	DefaultWSMaxConnections          = 10000
	DefaultWSMaxPayloadBytes         = 1 << 20
	DefaultWSWriteTimeout            = 10 * time.Second
	DefaultWSMaxSubscriptionsPerConn = 100
	DefaultWSMaxDroppedFrames        = 64
)

var ErrWSSubscriptionLimit = errors.New("subscription limit reached for this connection")

type WSConnection struct {
	CreatedAt time.Time
	Conn      *websocket.Conn

	lastSeen atomic.Int64
	dropped  atomic.Int64
	overflow atomic.Bool
	closed   atomic.Bool

	send chan WSPayload
	done chan struct{}
	ID   string
}

func (c *WSConnection) TrySend(payload WSPayload) bool {
	select {
	case c.send <- payload:
		return true
	default:
		c.dropped.Add(1)
		return false
	}
}

func (c *WSConnection) Closed() bool {
	return c.closed.Load()
}

func (c *WSConnection) Dropped() int64 {
	return c.dropped.Load()
}

func (c *WSConnection) LastSeenAt() time.Time {
	return time.Unix(0, c.lastSeen.Load())
}

type WSStats struct {
	Connections   int
	Subscriptions int
	Dropped       int64
	SlowConsumers int
}

type WSConnmgr struct {
	logger        common.Logger
	conns         map[string]*WSConnection
	subscriptions map[string]map[string]memorybroker.Subscription
	Broker        *memorybroker.Broker

	maxConns        int
	maxSubsPerConn  int
	maxDroppedFrame int64
	writeTimeout    time.Duration
	sendBufferSize  int

	onOverflow func(connID string, dropped int64)

	mu sync.RWMutex
}

func NewWSConnmgr(logger common.Logger, br *memorybroker.Broker) *WSConnmgr {
	var b memorybroker.Broker
	if br != nil {
		b = *br
	}
	if b == nil {
		b = memorybroker.NewMemoryBroker()
	}
	return &WSConnmgr{
		conns:           make(map[string]*WSConnection),
		subscriptions:   make(map[string]map[string]memorybroker.Subscription),
		Broker:          &b,
		logger:          logger,
		maxConns:        DefaultWSMaxConnections,
		maxSubsPerConn:  DefaultWSMaxSubscriptionsPerConn,
		maxDroppedFrame: DefaultWSMaxDroppedFrames,
		writeTimeout:    DefaultWSWriteTimeout,
		sendBufferSize:  DefaultWSSendBufferSize,
	}
}

func (connmgr *WSConnmgr) Register(connID string, wsConn *websocket.Conn) *WSConnection {
	connmgr.mu.Lock()
	defer connmgr.mu.Unlock()

	if _, exists := connmgr.conns[connID]; exists {
		return nil
	}
	if connmgr.maxConns > 0 && len(connmgr.conns) >= connmgr.maxConns {
		return nil
	}

	bufSize := connmgr.sendBufferSize
	if bufSize <= 0 {
		bufSize = DefaultWSSendBufferSize
	}

	now := time.Now()
	c := &WSConnection{
		ID:        connID,
		Conn:      wsConn,
		CreatedAt: now,
		send:      make(chan WSPayload, bufSize),
		done:      make(chan struct{}),
	}
	c.lastSeen.Store(now.UnixNano())
	connmgr.conns[connID] = c

	go writeLoop(wsConn, c.send, c.done, connmgr.writeTimeout, connmgr.logger)

	return c
}

func (connmgr *WSConnmgr) Unregister(connID string) {
	connmgr.mu.Lock()
	c, ok := connmgr.conns[connID]
	subs := connmgr.subscriptions[connID]
	delete(connmgr.subscriptions, connID)
	delete(connmgr.conns, connID)
	if ok {
		c.closed.Store(true)
		close(c.done)
	}
	connmgr.mu.Unlock()

	for _, sub := range subs {
		_ = (*connmgr.Broker).Unsubscribe(sub)
	}
}

func (connmgr *WSConnmgr) Get(connID string) (*WSConnection, bool) {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	c, ok := connmgr.conns[connID]
	return c, ok
}

func (connmgr *WSConnmgr) Count() int {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	return len(connmgr.conns)
}

func (connmgr *WSConnmgr) Stats() WSStats {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	stats := WSStats{Connections: len(connmgr.conns)}
	for _, subs := range connmgr.subscriptions {
		stats.Subscriptions += len(subs)
	}
	for _, c := range connmgr.conns {
		stats.Dropped += c.dropped.Load()
		if c.overflow.Load() {
			stats.SlowConsumers++
		}
	}

	return stats
}

func (connmgr *WSConnmgr) SubscriptionCount(connID string) int {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	return len(connmgr.subscriptions[connID])
}

func (connmgr *WSConnmgr) Topics(connID string) []string {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	topics := make([]string, 0, len(connmgr.subscriptions[connID]))
	for topic := range connmgr.subscriptions[connID] {
		topics = append(topics, topic)
	}

	return topics
}

func (connmgr *WSConnmgr) touch(connID string) {
	connmgr.mu.RLock()
	c, ok := connmgr.conns[connID]
	connmgr.mu.RUnlock()

	if ok {
		c.lastSeen.Store(time.Now().UnixNano())
	}
}

func (connmgr *WSConnmgr) trySend(c *WSConnection, payload WSPayload) bool {
	if c.TrySend(payload) {
		return true
	}

	dropped := c.dropped.Load()
	if c.overflow.CompareAndSwap(false, true) {
		if connmgr.logger != nil {
			connmgr.logger.Warn("WSSlowConsumer",
				"id", c.ID,
				"dropped", dropped,
				"reason", "send queue full; client is not draining the socket fast enough")
		}
		if connmgr.onOverflow != nil {
			connmgr.onOverflow(c.ID, dropped)
		}
	}

	if connmgr.maxDroppedFrame > 0 && dropped >= connmgr.maxDroppedFrame {
		if connmgr.logger != nil {
			connmgr.logger.Error("WSSlowConsumerEvicted",
				"id", c.ID,
				"dropped", dropped,
				"limit", connmgr.maxDroppedFrame)
		}
		connmgr.Evict(c.ID)
	}

	return false
}

func (connmgr *WSConnmgr) Evict(connID string) {
	connmgr.mu.Lock()
	c, ok := connmgr.conns[connID]
	if !ok {
		connmgr.mu.Unlock()
		return
	}
	subs := connmgr.subscriptions[connID]
	delete(connmgr.subscriptions, connID)
	delete(connmgr.conns, connID)
	c.closed.Store(true)
	close(c.done)
	connmgr.mu.Unlock()

	_ = c.Conn.Close()
	for _, sub := range subs {
		_ = (*connmgr.Broker).Unsubscribe(sub)
	}
}

func (connmgr *WSConnmgr) Subscribe(connID, topic string, handler memorybroker.MessageHandler) error {
	connmgr.mu.Lock()
	subs := connmgr.subscriptions[connID]
	if _, exists := subs[topic]; exists {
		connmgr.mu.Unlock()
		return nil
	}
	if connmgr.maxSubsPerConn > 0 && len(subs) >= connmgr.maxSubsPerConn {
		connmgr.mu.Unlock()
		return ErrWSSubscriptionLimit
	}
	connmgr.mu.Unlock()

	sub, err := (*connmgr.Broker).Subscribe(topic, handler)
	if err != nil {
		return err
	}

	connmgr.mu.Lock()
	if connmgr.subscriptions[connID] == nil {
		connmgr.subscriptions[connID] = make(map[string]memorybroker.Subscription, 4)
	}
	if _, exists := connmgr.subscriptions[connID][topic]; exists {
		connmgr.mu.Unlock()
		_ = (*connmgr.Broker).Unsubscribe(sub)
		return nil
	}
	connmgr.subscriptions[connID][topic] = sub
	connmgr.mu.Unlock()

	return nil
}

func (connmgr *WSConnmgr) isSubscribed(connID, topic string) bool {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	_, ok := connmgr.subscriptions[connID][topic]
	return ok
}

func (connmgr *WSConnmgr) Unsubscribe(connID, topic string) error {
	connmgr.mu.Lock()
	target, found := connmgr.subscriptions[connID][topic]
	if found {
		delete(connmgr.subscriptions[connID], topic)
		if len(connmgr.subscriptions[connID]) == 0 {
			delete(connmgr.subscriptions, connID)
		}
	}
	connmgr.mu.Unlock()

	if !found {
		return nil
	}

	return (*connmgr.Broker).Unsubscribe(target)
}

func (connmgr *WSConnmgr) startDeadConnDetection(interval, timeout time.Duration, done <-chan struct{}) {
	if done == nil {
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				connmgr.reapDeadConns(timeout)
			}
		}
	}()
}

func (connmgr *WSConnmgr) reapDeadConns(timeout time.Duration) []string {
	now := time.Now()

	connmgr.mu.RLock()
	var deadConnIDs []string
	for id, conn := range connmgr.conns {
		if now.Sub(conn.LastSeenAt()) > timeout {
			deadConnIDs = append(deadConnIDs, id)
		}
	}
	connmgr.mu.RUnlock()

	reaped := make([]string, 0, len(deadConnIDs))
	for _, id := range deadConnIDs {
		connmgr.mu.RLock()
		_, ok := connmgr.conns[id]
		connmgr.mu.RUnlock()
		if !ok {
			continue
		}

		if connmgr.logger != nil {
			connmgr.logger.Warn("WSHeartbeatTimeout", "id", id, "timeout", timeout.String())
		}
		connmgr.Evict(id)
		reaped = append(reaped, id)
	}

	return reaped
}
