package core

import (
	"sync"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/memorybroker"
	"golang.org/x/net/websocket"
)

const (
	sendBufferSize = 32

	DefaultWSMaxConnections  = 10000
	DefaultWSMaxPayloadBytes = 1 << 20
	DefaultWSWriteTimeout    = 10 * time.Second
)

type WSConnection struct {
	CreatedAt time.Time
	LastSeen  time.Time
	Conn      *websocket.Conn
	send      chan WSPayload
	done      chan struct{}
	ID        string
}

func (c *WSConnection) TrySend(payload WSPayload) bool {
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}

type WSConnmgr struct {
	logger        common.Logger
	conns         map[string]*WSConnection
	subscriptions map[string][]memorybroker.Subscription
	Broker        *memorybroker.Broker
	maxConns      int
	writeTimeout  time.Duration
	mu            sync.RWMutex
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
		conns:         make(map[string]*WSConnection),
		subscriptions: make(map[string][]memorybroker.Subscription),
		Broker:        &b,
		logger:        logger,
		maxConns:      DefaultWSMaxConnections,
		writeTimeout:  DefaultWSWriteTimeout,
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

	c := &WSConnection{
		ID:        connID,
		Conn:      wsConn,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		send:      make(chan WSPayload, sendBufferSize),
		done:      make(chan struct{}),
	}
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

func (connmgr *WSConnmgr) touch(connID string) {
	connmgr.mu.Lock()
	defer connmgr.mu.Unlock()

	if c, ok := connmgr.conns[connID]; ok {
		c.LastSeen = time.Now()
	}
}

func (connmgr *WSConnmgr) Subscribe(connID, topic string, handler memorybroker.MessageHandler) error {
	sub, err := (*connmgr.Broker).Subscribe(topic, handler)
	if err != nil {
		return err
	}

	connmgr.mu.Lock()
	connmgr.subscriptions[connID] = append(connmgr.subscriptions[connID], sub)
	connmgr.mu.Unlock()

	return nil
}

func (connmgr *WSConnmgr) isSubscribed(connID, topic string) bool {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	for _, sub := range connmgr.subscriptions[connID] {
		if sub.Topic() == topic {
			return true
		}
	}

	return false
}

func (connmgr *WSConnmgr) Unsubscribe(connID, topic string) error {
	connmgr.mu.RLock()
	var target memorybroker.Subscription
	found := false
	for _, sub := range connmgr.subscriptions[connID] {
		if sub.Topic() == topic {
			target = sub
			found = true
			break
		}
	}
	connmgr.mu.RUnlock()

	if !found {
		return nil
	}

	if err := (*connmgr.Broker).Unsubscribe(target); err != nil {
		return err
	}

	connmgr.mu.Lock()
	subs := connmgr.subscriptions[connID]
	for i, sub := range subs {
		if sub.Topic() == topic {
			connmgr.subscriptions[connID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	connmgr.mu.Unlock()

	return nil
}

func (connmgr *WSConnmgr) startDeadConnDetection(interval, timeout time.Duration, done <-chan struct{}) {
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

func (connmgr *WSConnmgr) reapDeadConns(timeout time.Duration) {
	now := time.Now()

	connmgr.mu.RLock()
	var deadConnIDs []string
	for id, conn := range connmgr.conns {
		if now.Sub(conn.LastSeen) > timeout {
			deadConnIDs = append(deadConnIDs, id)
		}
	}
	connmgr.mu.RUnlock()

	for _, id := range deadConnIDs {
		connmgr.mu.Lock()
		c, ok := connmgr.conns[id]
		if !ok {
			connmgr.mu.Unlock()
			continue
		}
		subs := connmgr.subscriptions[id]
		delete(connmgr.subscriptions, id)
		delete(connmgr.conns, id)
		close(c.done)
		connmgr.mu.Unlock()

		_ = c.Conn.Close()
		for _, sub := range subs {
			_ = (*connmgr.Broker).Unsubscribe(sub)
		}
	}
}
