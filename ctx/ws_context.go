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

	connID    string
	topic     string
	pattern   string
	operation string

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
	c.connID = ""
	c.topic = ""
	c.pattern = ""
	c.operation = ""
}

// SetSend wires the function Send delivers data through. ctx has no
// connection-send capability of its own (that lives in core, to avoid an
// import cycle) — the framework rebinds this per dispatch phase, e.g. to
// reply with an error payload while running an exception filter's Catch.
func (c *WSContext) SetSend(fn func(data any)) {
	c.send = fn
}

// Send delivers data back to the client through whatever the framework
// wired up for the current dispatch phase. No-op if nothing is wired.
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

func (c *WSContext) SetEvent(connID, pattern, topic, operation string) {
	c.connID = connID
	c.pattern = pattern
	c.topic = topic
	c.operation = operation
}

func (c *WSContext) ConnID() string {
	return c.connID
}

func (c *WSContext) Topic() string {
	return c.topic
}

func (c *WSContext) Pattern() string {
	return c.pattern
}

func (c *WSContext) Operation() string {
	return c.operation
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
