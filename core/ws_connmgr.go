package core

import (
	"sync"
	"time"

	"github.com/dangduoc08/ginject/matcher"
	"golang.org/x/net/websocket"
)

type Connection struct {
	ClientID  string
	WSConn    *websocket.Conn
	CreatedAt time.Time
	Events    []matcher.Pattern
}

type WSConnmgr struct {
	// Storage memcache.MemoryCache
	Conns map[string]*Connection
	rwMu  *sync.RWMutex
}

// func NewConnectionManager() *ConnectionManager {

// 	return &ConnectionManager{
// 		Storage: cache.NewMemoryCache(),
// 		Conns:   map[string]*Connection{},
// 		rwMu:    &sync.RWMutex{},
// 	}
// }

// func (connmgr *ConnectionManager) Register(clientID string, events []matcher.Pattern, wsConn *websocket.Conn) {
// 	connmgr.rwMu.Lock()

// 	defer connmgr.rwMu.Unlock()

// 	connmgr.Conns[clientID] = &Connection{
// 		ClientID:  clientID,
// 		NodeID:    runtime.NodeID(),
// 		WSConn:    wsConn,
// 		CreatedAt: time.Now().UTC(),
// 		Events:    events,
// 	}
// }
