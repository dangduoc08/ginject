package ctx

import (
	"context"
	"time"

	"golang.org/x/net/websocket"
)

type WSContext struct {
	*websocket.Conn

	contextID
	ctx context.Context

	Next Next

	payload   WSPayload
	Timestamp time.Time

	send func(data any)
}

func NewWSContext() *WSContext {
	return &WSContext{}
}

func (c *WSContext) Init(conn *websocket.Conn) {
	c.Timestamp = time.Now()
	c.Conn = conn
	c.SetID()
}

func (c *WSContext) Reset() {
	c.id = ""
	c.ctx = nil
	c.Next = nil
	c.Conn = nil
	c.payload = nil
	c.send = nil
}

func (c *WSContext) SetSend(fn func(data any)) {
	c.send = fn
}

func (c *WSContext) Send(data any) {
	if c.send != nil {
		c.send(data)
	}
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
