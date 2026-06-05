package connmgr

import (
	"context"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/internal/crypto"
	"golang.org/x/net/websocket"
)

const sendBufferSize = 256

type Connection struct {
	ID        string
	UserID    string
	CreatedAt time.Time

	conn      *websocket.Conn
	send      chan []byte
	done      chan struct{}
	startOnce sync.Once
	closeOnce sync.Once
}

func NewConnection(conn *websocket.Conn, userID string) (*Connection, error) {
	id, err := crypto.UUID()
	if err != nil {
		return nil, err
	}
	return &Connection{
		ID:        id,
		UserID:    userID,
		CreatedAt: time.Now(),
		conn:      conn,
		send:      make(chan []byte, sendBufferSize),
		done:      make(chan struct{}),
	}, nil
}

// Send enqueues msg for the write loop without blocking.
//
// It uses a three-way select so that:
//   - if the connection is still open and the buffer has space, the message is
//     enqueued (returns true);
//   - if done is closed (connection closing), returns false immediately;
//   - if the buffer is full, returns false immediately (message dropped).
//
// The send channel is never closed, which prevents a panic if two goroutines
// race on Send and Close. The done channel is the sole lifecycle signal.
func (c *Connection) Send(msg []byte) bool {
	select {
	case c.send <- msg:
		return true
	case <-c.done:
		return false
	default:
		return false
	}
}

func (c *Connection) Start(ctx context.Context) {
	c.startOnce.Do(func() {
		go c.writeLoop(ctx)
	})
}

func (c *Connection) writeLoop(ctx context.Context) {
	defer c.Close()
	for {
		select {
		case msg := <-c.send:
			if err := websocket.Message.Send(c.conn, msg); err != nil {
				return
			}
		case <-ctx.Done():
			return
		case <-c.done:
			return
		}
	}
}

func (c *Connection) Close() {
	c.closeOnce.Do(func() {
		close(c.done)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

func (c *Connection) Done() <-chan struct{} {
	return c.done
}

func (c *Connection) Conn() *websocket.Conn {
	return c.conn
}
