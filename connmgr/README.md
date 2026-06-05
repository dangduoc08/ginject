# connmgr

Connection Manager for the Ginject WebSocket runtime.

Answers one question: **what WebSocket connections are currently alive?**

It does not manage event subscriptions, rooms, presence, authorization, routing, or broker logic. Those are separate managers.

---

## Folder structure

```
connmgr/
├── connection.go            — Connection type and write loop
├── connection_test.go       — Connection unit + integration tests
├── manager.go               — ConnectionManager
├── manager_test.go          — ConnectionManager tests
└── manager_bench_test.go    — Benchmarks
```

---

## Public API

### Connection

```go
type Connection struct {
    ID        string
    UserID    string    // empty for anonymous connections
    CreatedAt time.Time
}

// NewConnection creates a Connection wrapping the given websocket. userID may
// be empty for anonymous connections.
func NewConnection(conn *websocket.Conn, userID string) *Connection

// Send enqueues msg for delivery. Non-blocking: returns false if the internal
// write buffer (256 slots) is full or the connection is closing.
func (c *Connection) Send(msg []byte) bool

// Start launches the write loop goroutine. The loop exits when ctx is
// cancelled or Close is called. Calling Start more than once is a no-op.
func (c *Connection) Start(ctx context.Context)

// Close closes the connection idempotently. Safe to call from multiple
// goroutines.
func (c *Connection) Close()

// Done returns a channel that is closed when the connection has been closed.
func (c *Connection) Done() <-chan struct{}

// Conn returns the underlying *websocket.Conn for use in read loops.
// Do not write to it directly; use Send.
func (c *Connection) Conn() *websocket.Conn
```

### ConnectionManager

```go
func NewConnectionManager() *ConnectionManager

func (m *ConnectionManager) Add(conn *Connection)
func (m *ConnectionManager) Remove(connID string)
func (m *ConnectionManager) Get(connID string) (*Connection, bool)
func (m *ConnectionManager) Exists(connID string) bool
func (m *ConnectionManager) Count() int                          // O(1), lock-free
func (m *ConnectionManager) Connections() []*Connection          // snapshot
func (m *ConnectionManager) GetByUser(userID string) []*Connection  // O(k) where k = user's connection count
```

All operations are concurrency-safe.

---

## Lifecycle

```
Handshake
    ↓
conn := NewConnection(wsConn, userID)
    ↓
manager.Add(conn)
    ↓
conn.Start(ctx)        ← write loop goroutine starts
    ↓
Read loop (caller)     ← reads from conn.Conn()
    ↓
conn.Send(msg)         ← any goroutine may call concurrently
    ↓
Disconnect (EOF / ctx cancel / write error)
    ↓
conn.Close()           ← idempotent; called by write loop or caller
    ↓
manager.Remove(conn.ID)
```

The caller owns the read loop. The `Connection` owns the write loop.

---

## Concurrency design

| Shared state | Protection | Reason |
|---|---|---|
| `ConnectionManager.byID`, `byUser` | `sync.RWMutex` | Multiple concurrent readers, rare writes |
| `ConnectionManager.count` | `atomic.Int64` | `Count()` hot path reads without holding the mutex |
| `Connection.send` | buffered channel (256) | Write loop is the sole reader; any goroutine may send |
| `Connection.closing` | `atomic.Bool` | Guards `Send()` against sending on a logically-closed connection without closing the channel |
| `Connection.done` | closed once via `sync.Once` | Signals all `Done()` waiters and the write loop atomically |

**Why the write loop pattern?**

The `golang.org/x/net/websocket` `Conn` type serializes writes internally, so concurrent `websocket.Message.Send` calls won't corrupt the wire. However, the write loop still provides:

1. **Ordering** — messages from multiple goroutines are delivered in the order they were enqueued.
2. **Back-pressure** — if the remote is slow, the send buffer fills up and `Send()` returns `false` instead of blocking the caller.
3. **Clean shutdown** — `Close()` signals the loop via `done`; no goroutine leaks.

**Why not close the `send` channel in `Close()`?**

Closing a channel while goroutines are still sending to it causes a panic. Since `Send()` can be called from any goroutine, the channel must remain open. Instead, `closing` is set atomically so `Send()` refuses new messages, and the write loop exits via `<-done`.

---

## Benchmarks (amd64, i7-9750H)

| Benchmark | ns/op | allocs/op |
|---|---|---|
| `Add` | 1 205 | 0 |
| `Get` | 26 | 0 |
| `Remove` | 1 116 | 0 |
| `Count` | 0.4 | 0 |
| `Connections` (1 000 conns) | 20 748 | 1 |
| `GetByUser` (100 conns/user) | 2 090 | 1 |
| Parallel `Get` | 66 | 0 |

---

## Future extensions

| Module | Integration point |
|---|---|
| `SubscriptionManager` | Holds a `*ConnectionManager`; calls `Get` to resolve conn from ID |
| `RoomManager` | Same; builds its own `roomID → []connID` index on top |
| `PresenceManager` | Watches `conn.Done()` to detect disconnects |
| `Inspector` | Calls `Count()` and `Connections()` for dashboards |
| `Tracing` | Reads `Connection.ID` and `Connection.CreatedAt` as span attributes |
| `Cluster` | Connection IDs are UUIDs — globally unique; remote nodes hold proxy `Connection` objects |

No breaking changes are required for any of these: all consume the existing API.
