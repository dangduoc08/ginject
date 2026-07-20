package ctx

import (
	"context"
	"time"

	"github.com/dangduoc08/ginject/internal/crypto"
	"golang.org/x/net/websocket"
)

type WSContext struct {
	*websocket.Conn

	id  string
	ctx context.Context

	Next  Next
	Event *Event

	payload   WSPayload
	Timestamp time.Time
}

func NewWSContext() *WSContext {
	return &WSContext{
		Event: NewEvent(),
	}
}

func (c *WSContext) Init(conn *websocket.Conn) {
	c.Timestamp = time.Now()
	c.Conn = conn
}

func (c *WSContext) Reset() {
	c.id = ""
	c.ctx = nil
	c.Next = nil
	c.Conn = nil
	c.payload = nil
	c.Event.reset()
}

func (c *WSContext) WSPayload() WSPayload {
	return c.payload
}

func (c *WSContext) SetWSPayload(p WSPayload) {
	c.payload = p
}

func (c *WSContext) Context() context.Context {
	if c.ctx == nil {
		return context.Background()
	}
	return c.ctx
}

func (c *WSContext) SetContext(ctx context.Context) {
	c.ctx = ctx
}

func (c *WSContext) SetID() {
	if c.id == "" {
		c.id, _ = crypto.UUID()
	}
}

func (c *WSContext) GetID() string {
	return c.id
}
