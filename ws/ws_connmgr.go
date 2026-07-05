package ws

import (
	"sync"
	"time"

	"github.com/dangduoc08/ginject/broker2"
	"golang.org/x/net/websocket"
)

type WSConnection struct {
	ClientID  string
	WSConn    *websocket.Conn
	CreatedAt time.Time
}

type WSConnmgr struct {
	Conns  map[string]*WSConnection
	Broker broker2.Broker
	rwMu   *sync.RWMutex
}

func NewWSConnmgr() *WSConnmgr {

	return &WSConnmgr{
		rwMu:   &sync.RWMutex{},
		Broker: broker2.NewBroker(),
		Conns:  map[string]*WSConnection{},
	}
}

func (connmgr *WSConnmgr) Register(clientID string, wsConn *websocket.Conn) {
	connmgr.rwMu.Lock()
	defer connmgr.rwMu.Unlock()

	if _, ok := connmgr.Conns[clientID]; !ok {
		connmgr.Conns[clientID] = &WSConnection{
			ClientID:  clientID,
			WSConn:    wsConn,
			CreatedAt: time.Now(),
		}
	}
}
