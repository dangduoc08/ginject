package ctx

import (
	"context"
	"sync"
	"time"
)

// Protocol identifies the transport type that owns this execution.
type Protocol string

const (
	ProtoHTTP Protocol = "http"
	ProtoWS   Protocol = "ws"
	ProtoRPC  Protocol = "rpc"
	ProtoGQL  Protocol = "gql"
)

type metaKey string

// Well-known metadata keys. Use these constants instead of raw strings.
const (
	MetaRequestID metaKey = "requestID"
	MetaTraceID   metaKey = "traceID"
	MetaUser      metaKey = "user"
)

// ExecutionContext is the shared runtime control plane for a single
// request or WebSocket connection lifetime.
//
// It embeds context.Context so every stdlib and third-party library
// that accepts context.Context works transparently.
//
// Rules:
//   - Pass explicitly as the first parameter through handlers, middleware,
//     guards, and interceptors.
//   - Never store in long-lived structs (modules, globals, WS state).
//   - Always defer the paired cancel() returned by the constructor.
type ExecutionContext struct {
	context.Context
	mu       sync.RWMutex
	meta     map[metaKey]any
	protocol Protocol
}

// NewHTTPExecutionContext creates an ExecutionContext rooted at the
// incoming HTTP request context so client-disconnect cancels it, with
// an additional hard timeout deadline for SLA enforcement.
// Caller must defer cancel().
func NewHTTPExecutionContext(parent context.Context, timeout time.Duration) (*ExecutionContext, context.CancelFunc) {
	tctx, cancel := context.WithTimeout(parent, timeout)
	return &ExecutionContext{
		Context:  tctx,
		meta:     make(map[metaKey]any, 4),
		protocol: ProtoHTTP,
	}, cancel
}

// NewWSExecutionContext creates an ExecutionContext rooted at
// context.Background(). It must NOT be derived from the HTTP upgrade
// request context: that context is cancelled when the upgrade
// handshake completes.
// Caller must defer cancel() on connection close.
func NewWSExecutionContext() (*ExecutionContext, context.CancelFunc) {
	bctx, cancel := context.WithCancel(context.Background())
	return &ExecutionContext{
		Context:  bctx,
		meta:     make(map[metaKey]any, 4),
		protocol: ProtoWS,
	}, cancel
}

// WithTimeout derives a child ExecutionContext with an additional SLA
// deadline layered on top of the parent's existing deadline.
// Metadata is snapshot-copied so the child inherits the parent's
// values without sharing the mutex.
// Caller must defer cancel().
func WithTimeout(parent *ExecutionContext, d time.Duration) (*ExecutionContext, context.CancelFunc) {
	child, cancel := context.WithTimeout(parent.Context, d)

	parent.mu.RLock()
	snapshot := make(map[metaKey]any, len(parent.meta))
	for k, v := range parent.meta {
		snapshot[k] = v
	}
	parent.mu.RUnlock()

	return &ExecutionContext{
		Context:  child,
		meta:     snapshot,
		protocol: parent.protocol,
	}, cancel
}

// Protocol returns the transport type that owns this execution.
func (e *ExecutionContext) Protocol() Protocol { return e.protocol }

// Set stores a request-scoped key/value pair. Safe for concurrent use.
func (e *ExecutionContext) Set(key metaKey, val any) *ExecutionContext {
	e.mu.Lock()
	e.meta[key] = val
	e.mu.Unlock()
	return e
}

// Get reads a request-scoped value. Safe for concurrent use.
func (e *ExecutionContext) Get(key metaKey) (any, bool) {
	e.mu.RLock()
	v, ok := e.meta[key]
	e.mu.RUnlock()
	return v, ok
}

// MustGet returns the value for key or panics if absent.
// Use only when absence is a programming error guaranteed by a
// middleware contract (e.g. JWT guard always sets MetaUser).
func (e *ExecutionContext) MustGet(key metaKey) any {
	v, ok := e.Get(key)
	if !ok {
		panic("ExecutionContext: missing required key: " + string(key))
	}
	return v
}
