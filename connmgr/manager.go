package connmgr

import (
	"sync"
	"sync/atomic"
)

type ConnectionManager struct {
	mu     sync.RWMutex
	byID   map[string]*Connection
	byUser map[string]map[string]*Connection
	count  atomic.Int64
}

func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		byID:   make(map[string]*Connection),
		byUser: make(map[string]map[string]*Connection),
	}
}

func (m *ConnectionManager) Add(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byID[conn.ID]; exists {
		return
	}
	m.byID[conn.ID] = conn
	if conn.UserID != "" {
		if m.byUser[conn.UserID] == nil {
			m.byUser[conn.UserID] = make(map[string]*Connection)
		}
		m.byUser[conn.UserID][conn.ID] = conn
	}
	m.count.Add(1)
}

func (m *ConnectionManager) Remove(connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conn, ok := m.byID[connID]
	if !ok {
		return
	}
	delete(m.byID, connID)
	if conn.UserID != "" {
		userConns := m.byUser[conn.UserID]
		delete(userConns, connID)
		if len(userConns) == 0 {
			delete(m.byUser, conn.UserID)
		}
	}
	m.count.Add(-1)
}

func (m *ConnectionManager) Get(connID string) (*Connection, bool) {
	m.mu.RLock()
	conn, ok := m.byID[connID]
	m.mu.RUnlock()
	return conn, ok
}

func (m *ConnectionManager) Exists(connID string) bool {
	m.mu.RLock()
	_, ok := m.byID[connID]
	m.mu.RUnlock()
	return ok
}

func (m *ConnectionManager) Count() int {
	return int(m.count.Load())
}

func (m *ConnectionManager) Connections() []*Connection {
	m.mu.RLock()
	out := make([]*Connection, 0, len(m.byID))
	for _, c := range m.byID {
		out = append(out, c)
	}
	m.mu.RUnlock()
	return out
}

func (m *ConnectionManager) GetByUser(userID string) []*Connection {
	m.mu.RLock()
	userConns := m.byUser[userID]
	out := make([]*Connection, 0, len(userConns))
	for _, c := range userConns {
		out = append(out, c)
	}
	m.mu.RUnlock()
	return out
}
