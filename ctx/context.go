package ctx

import "github.com/dangduoc08/ginject/internal/crypto"

type Context interface {
	Reset()
	SetID()
	GetID() string
}

type contextID struct {
	id string
}

func (c *contextID) SetID() {
	if c.id == "" {
		c.id, _ = crypto.UUID()
	}
}

func (c *contextID) GetID() string {
	return c.id
}

type (
	Map         map[string]any
	ErrFunc     func(error)
	HTTPHandler = func(*HTTPContext)
	WSHandler   = func(*WSContext)
	Next        = func()
	Redirect    = func(string)
)
