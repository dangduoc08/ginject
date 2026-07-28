package core

import (
	"sync"
	"time"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/common"
	"golang.org/x/net/websocket"
)

const sendBufferSize = 32

type WSConnection struct {
	ID        string
	Conn      *websocket.Conn
	CreatedAt time.Time
	LastSeen  time.Time

	send chan WSPayload
	done chan struct{}
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
	mu            sync.RWMutex
	conns         map[string]*WSConnection
	subscriptions map[string][]broker.Subscription

	Broker *broker.Broker
	logger common.Logger
}

func NewWSConnmgr(logger common.Logger, br *broker.Broker) *WSConnmgr {
	var b broker.Broker
	if br != nil {
		b = *br
	}
	if b == nil {
		b = broker.NewBroker()
	}
	return &WSConnmgr{
		conns:         make(map[string]*WSConnection),
		subscriptions: make(map[string][]broker.Subscription),
		Broker:        &b,
		logger:        logger,
	}
}

func (connmgr *WSConnmgr) Register(connID string, wsConn *websocket.Conn) *WSConnection {
	connmgr.mu.Lock()
	defer connmgr.mu.Unlock()

	c := &WSConnection{
		ID:        connID,
		Conn:      wsConn,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
		send:      make(chan WSPayload, sendBufferSize),
		done:      make(chan struct{}),
	}
	connmgr.conns[connID] = c

	go writeLoop(wsConn, c.send, c.done, connmgr.logger)

	return c
}

func (connmgr *WSConnmgr) Unregister(connID string) {
	connmgr.mu.Lock()
	defer connmgr.mu.Unlock()

	if c, ok := connmgr.conns[connID]; ok {
		close(c.done)
	}

	for _, sub := range connmgr.subscriptions[connID] {
		_ = (*connmgr.Broker).Unsubscribe(sub)
	}
	delete(connmgr.subscriptions, connID)
	delete(connmgr.conns, connID)
}

func (connmgr *WSConnmgr) Get(connID string) (*WSConnection, bool) {
	connmgr.mu.RLock()
	defer connmgr.mu.RUnlock()

	c, ok := connmgr.conns[connID]
	return c, ok
}

func (connmgr *WSConnmgr) touch(connID string) {
	connmgr.mu.Lock()
	defer connmgr.mu.Unlock()

	if c, ok := connmgr.conns[connID]; ok {
		c.LastSeen = time.Now()
	}
}

func (connmgr *WSConnmgr) Subscribe(connID, topic string, handler broker.MessageHandler) error {
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
	connmgr.mu.Lock()
	defer connmgr.mu.Unlock()

	subs := connmgr.subscriptions[connID]
	for i, sub := range subs {
		if sub.Topic() != topic {
			continue
		}

		if err := (*connmgr.Broker).Unsubscribe(sub); err != nil {
			return err
		}

		connmgr.subscriptions[connID] = append(subs[:i], subs[i+1:]...)
		return nil
	}

	return nil
}
