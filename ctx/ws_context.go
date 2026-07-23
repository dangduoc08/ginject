package ctx

import (
	"context"
	"time"

	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/internal/crypto"
	"golang.org/x/net/websocket"
)

type WSContext struct {
	*websocket.Conn

	id  string
	ctx context.Context

	Next  Next
	Event *event.Event

	payload   WSPayload
	Timestamp time.Time

	send func(data any)
}

func NewWSContext() *WSContext {
	return &WSContext{
		Event: event.NewEvent(),
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
	c.send = nil
	c.Event.Reset()
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
