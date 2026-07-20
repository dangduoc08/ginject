package ctx

import (
	"context"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
	"golang.org/x/net/websocket"
)

func TestNewWSContext(t *testing.T) {
	c := NewWSContext()
	if c.Event == nil {
		t.Error(test.DiffMessage(c.Event, "<non-nil>", "NewWSContext should initialize Event"))
	}
}

func TestWSContext_Init(t *testing.T) {
	c := NewWSContext()
	conn := &websocket.Conn{}
	c.Init(conn)

	if c.Conn != conn {
		t.Error(test.DiffMessage(c.Conn, conn, "Init should store the given connection"))
	}
	if c.Timestamp.IsZero() {
		t.Error(test.DiffMessage(c.Timestamp.IsZero(), false, "Init should set Timestamp"))
	}
}

func TestWSContext_SetID(t *testing.T) {
	c := NewWSContext()
	c.SetID()
	if c.id == "" {
		t.Error(test.DiffMessage(c.id, "<non-empty UUID>", "SetID should generate an id"))
	}
}

func TestWSContext_SetIDIdempotent(t *testing.T) {
	c := NewWSContext()
	c.SetID()
	first := c.id
	c.SetID()
	if c.id != first {
		t.Error(test.DiffMessage(c.id, first, "SetID should not overwrite an existing id"))
	}
}

func TestWSContext_GetID(t *testing.T) {
	c := NewWSContext()
	c.SetID()
	if c.GetID() != c.id {
		t.Error(test.DiffMessage(c.GetID(), c.id, "GetID should return the id"))
	}
}

func TestWSContext_WSPayload(t *testing.T) {
	c := NewWSContext()
	if c.WSPayload() != nil {
		t.Error(test.DiffMessage(c.WSPayload(), nil, "WSPayload should be nil before SetWSPayload"))
	}
	p := WSPayload{"foo": "bar"}
	c.SetWSPayload(p)
	if c.WSPayload()["foo"] != "bar" {
		t.Error(test.DiffMessage(c.WSPayload(), p, "WSPayload should return what SetWSPayload stored"))
	}
}

func TestWSContext_ContextDefaultsToBackground(t *testing.T) {
	c := NewWSContext()
	if c.Context() != context.Background() {
		t.Error(test.DiffMessage(c.Context(), context.Background(), "Context should default to context.Background()"))
	}
}

func TestWSContext_SetContext(t *testing.T) {
	c := NewWSContext()
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "v")
	c.SetContext(ctx)
	if c.Context() != ctx {
		t.Error(test.DiffMessage(c.Context(), ctx, "Context should return the context set by SetContext"))
	}
}

func TestWSContext_Reset(t *testing.T) {
	c := NewWSContext()
	type key struct{}
	c.SetID()
	c.SetContext(context.WithValue(context.Background(), key{}, "v"))
	c.SetWSPayload(WSPayload{"foo": "bar"})
	c.Next = func() {}

	c.Reset()

	if c.id != "" {
		t.Error(test.DiffMessage(c.id, "", "Reset should clear id"))
	}
	if c.ctx != nil {
		t.Error(test.DiffMessage(c.ctx, nil, "Reset should clear ctx"))
	}
	if c.Conn != nil {
		t.Error(test.DiffMessage(c.Conn, nil, "Reset should clear Conn"))
	}
	if c.payload != nil {
		t.Error(test.DiffMessage(c.payload, nil, "Reset should clear payload"))
	}
	if c.Next != nil {
		t.Error(test.DiffMessage(c.Next, nil, "Reset should clear Next"))
	}
	if c.Context() != context.Background() {
		t.Error(test.DiffMessage(c.Context(), context.Background(), "Context should fall back to Background after Reset"))
	}
}
