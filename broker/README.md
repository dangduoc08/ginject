# broker

Production-grade in-memory event broker (pub/sub) for the Ginject framework.

---

## Packages

| Package | Path | Role |
|---|---|---|
| `broker` | `github.com/dangduoc08/ginject/broker` | Broker interface, `MemoryBroker`, pub/sub logic |
| `matcher` | `github.com/dangduoc08/ginject/matcher` | Topic pattern parsing and matching — no broker dependency |

---

## Quick start

```go
import "github.com/dangduoc08/ginject/broker"

b := broker.New()
defer b.Close()

sub, _ := b.Subscribe("user.created", func(m *broker.Message) {
    fmt.Printf("new user: %v\n", m.Payload)
})

b.Publish("user.created", map[string]string{"name": "alice"})

sub.Unsubscribe()
```

---

## Constructors

```go
// New returns a broker with safe defaults:
//   - RecoverPanics: true
//   - AsyncWorkers:  runtime.GOMAXPROCS(0)
//   - AsyncQueueSize: workers * 64
func New() Broker

// NewWithConfig returns a broker with custom configuration.
func NewWithConfig(cfg Config) Broker
```

---

## Config

```go
type Config struct {
    // Recover panics inside handlers so one misbehaving handler cannot
    // abort delivery to the rest. Default true via New().
    RecoverPanics bool

    // Called when a handler panics and RecoverPanics is true.
    // If OnPanic itself panics, the secondary panic is silently discarded.
    OnPanic func(*Message, any)

    // Number of background workers for PublishAsync.
    // 0 → PublishAsync returns ErrNoAsyncWorkers.
    AsyncWorkers int

    // Capacity of the async job channel. Defaults to AsyncWorkers * 64.
    AsyncQueueSize int

    // Observability hooks. All four are nil-safe: a panicking hook does not
    // abort delivery or affect other hooks.
    BeforePublish  func(topic string, payload any)
    AfterPublish   func(topic string, payload any, err error)
    BeforeDispatch func(msg *Message, handler int)
    AfterDispatch  func(msg *Message, handler int)
}
```

---

## Types

```go
type Message struct {
    ID        string         // UUID v4, generated at Publish time
    Topic     string         // exact topic string that was published
    Payload   any
    Timestamp time.Time
    Metadata  map[string]any // caller-managed; nil unless populated externally
}

type MessageHandler func(*Message)

type Subscription interface {
    ID() string
    Topic() string         // the pattern string passed to Subscribe/Once/SubscribeQueue
    Unsubscribe() error
}

type Stats struct {
    Topics          int
    Subscribers     int
    MessagesSent    uint64
    MessagesDropped uint64  // incremented on PublishAsync queue-full
    PublishCalls    uint64
}
```

---

## Broker interface

```go
type Broker interface {
    Publish(topic string, payload any) error
    PublishAsync(topic string, payload any) error
    Subscribe(topic string, handler MessageHandler) (Subscription, error)
    Once(topic string, handler MessageHandler) (Subscription, error)
    SubscribeQueue(topic, group string, handler MessageHandler) (Subscription, error)
    Unsubscribe(sub Subscription) error
    Off(topic string) error
    ListenerCount(topic string) int
    Topics() []string
    Clear() error
    Close() error
    Stats() Stats
}
```

---

## Sentinel errors

```go
var (
    ErrClosed          // broker is closed
    ErrNilHandler      // handler argument is nil
    ErrEmptyTopic      // topic argument is empty string
    ErrEmptyGroup      // group argument is empty string (SubscribeQueue)
    ErrAsyncQueueFull  // PublishAsync worker queue is full
    ErrNoAsyncWorkers  // PublishAsync called but AsyncWorkers == 0
    ErrWildcardInQueue // SubscribeQueue called with a wildcard topic
)
```

---

## Topic wildcard patterns

Patterns are parsed **once at Subscribe time** by the `matcher` package. Publish performs O(1) lookup for exact, single-suffix, and global patterns. Complex patterns require O(patterns) scan.

| Pattern | Kind | Example matches | Example non-matches |
|---|---|---|---|
| `user.created` | Exact | `user.created` | `user.updated` |
| `*` | Global | every topic | — |
| `>` | Global | every topic | — |
| `user.*` | Single-suffix | `user.created`, `user.deleted` | `user.profile.updated` |
| `user.>` | Complex | `user.created`, `user.profile.updated`, `user.a.b.c` | `user` |
| `tenant.*.user.created` | Complex | `tenant.1.user.created`, `tenant.abc.user.created` | `tenant.1.user.updated` |
| `tenant.*.user.>` | Complex | `tenant.1.user.created`, `tenant.1.user.profile.updated` | `tenant.1.admin.x` |
| `*.created` | Complex | `user.created`, `order.created` | `a.b.created` |

**Backward compatibility:** `*` is an alias for `>` (matches all topics). `user.*` continues to match only one level below `user.`.

**`SubscribeQueue` requires an exact topic.** Wildcard patterns return `ErrWildcardInQueue`.

---

## Delivery patterns

### 1. Fan-Out — all subscribers receive

```go
b.Subscribe("order.created", notifyEmail)
b.Subscribe("order.created", notifySlack)
b.Subscribe("order.created", updateInventory)

b.Publish("order.created", order) // all three handlers fire
```

### 2. Direct / Point-to-Point — unique topic per connection

```go
// Each client subscribes to its own topic.
b.Subscribe("conn."+connID, func(m *broker.Message) {
    ws.WriteJSON(m.Payload)
})

// Send to exactly one client.
b.Publish("conn.abc123", payload)
```

### 3. Room / Group — shared topic

```go
b.Subscribe("room.42", clientA)
b.Subscribe("room.42", clientB)

b.Publish("room.42", chatMsg) // both clients receive
```

### 4. Queue / Competing Consumer — one handler per group

```go
for i := 0; i < 4; i++ {
    i := i
    b.SubscribeQueue("task.process", "workers", func(m *broker.Message) {
        fmt.Printf("worker %d: %v\n", i, m.Payload)
    })
}

b.Publish("task.process", job1) // → worker 0
b.Publish("task.process", job2) // → worker 1 (round-robin)
b.Publish("task.process", job3) // → worker 2
```

Multiple independent groups each receive one delivery:

```go
b.SubscribeQueue("order.created", "billing",   billingHandler)
b.SubscribeQueue("order.created", "inventory", inventoryHandler)

b.Publish("order.created", order)
// billing receives once AND inventory receives once — independently
```

Fan-out and queue coexist on the same topic:

```go
b.Subscribe("order.created", auditLog)             // always fires
b.SubscribeQueue("order.created", "billing", h1)   // exactly one of h1/h2 fires
b.SubscribeQueue("order.created", "billing", h2)

b.Publish("order.created", order)
```

### 5. Broadcast — global wildcard

```go
b.Subscribe("*", func(m *broker.Message) {
    fmt.Printf("[audit] %s %v\n", m.Topic, m.Payload)
})

b.Publish("anything.at.all", data) // audit handler always fires
```

---

## Once — one-shot delivery

```go
b.Once("app.started", func(m *broker.Message) {
    fmt.Println("startup hook fired")
})

b.Publish("app.started", nil) // fires
b.Publish("app.started", nil) // no-op — subscription was removed
```

`Once` is safe under concurrent `Publish`: the handler fires **at most once** even when multiple goroutines publish simultaneously (guarded by `atomic.Bool`).

---

## Unsubscribe

```go
sub, _ := b.Subscribe("event", handler)

sub.Unsubscribe()       // via Subscription interface
b.Unsubscribe(sub)      // or via Broker

b.Off("event")          // remove all handlers registered under this exact pattern
b.Off("user.*")         // remove all handlers for the "user.*" pattern
b.Off("*")              // remove all global handlers
```

---

## Asynchronous publish

`PublishAsync` requires `AsyncWorkers > 0` (set in `Config` or provided automatically by `New()`).

```go
b.PublishAsync("metrics.collected", stats)
```

Jobs are sent to a bounded worker pool. When the queue is full, `ErrAsyncQueueFull` is returned immediately — no blocking, no goroutine explosion.

```go
b := broker.NewWithConfig(broker.Config{
    RecoverPanics:  true,
    AsyncWorkers:   8,
    AsyncQueueSize: 1024,
})

err := b.PublishAsync("log.event", entry)
if errors.Is(err, broker.ErrAsyncQueueFull) {
    // drop or fallback
}
```

| | `Publish` | `PublishAsync` |
|---|---|---|
| Error propagation | Yes | No (fire-and-forget) |
| Delivery guarantee | Per-call | Best-effort |
| Caller blocking | Until all handlers return | Returns immediately |
| Use case | Business logic, ordering | Metrics, audit, notifications |

---

## Observability hooks

```go
b := broker.NewWithConfig(broker.Config{
    RecoverPanics: true,
    BeforePublish: func(topic string, _ any) {
        fmt.Println("publishing to", topic)
    },
    AfterPublish: func(topic string, _ any, err error) {
        metrics.Inc("broker.publish", topic)
    },
    BeforeDispatch: func(msg *broker.Message, i int) {
        span.AddEvent("dispatch", msg.Topic, i)
    },
    AfterDispatch: func(msg *broker.Message, i int) {
        span.End()
    },
    OnPanic: func(msg *broker.Message, r any) {
        log.Error("handler panic", "topic", msg.Topic, "recovered", r)
    },
})
```

Hook panics are **isolated**: a panicking hook does not abort delivery, does not affect other hooks, and does not affect `OnPanic`. All four hooks are nil-safe and zero-cost when disabled.

---

## Stats

```go
s := b.Stats()
fmt.Println(s.Topics, s.Subscribers, s.MessagesSent, s.PublishCalls)
```

Counters are `atomic.Uint64` — safe to read from any goroutine without locking.

---

## Close

```go
b.Close()
```

Marks the broker closed (`ErrClosed` on all subsequent calls), drains and shuts down the async worker pool (if configured), then clears all subscription maps. Safe to call from any goroutine.

---

## Concurrency model

- A single `sync.RWMutex` protects all subscription maps (`exact`, `prefix`, `global`, `complex`, `queueGroups`). No nested locks.
- `Publish` acquires `RLock` only long enough to snapshot handler references, then releases before invoking any handler. This prevents deadlock when a handler re-enters the broker.
- `PublishAsync` vs `Close` race is eliminated via a separate `closeMu sync.RWMutex`: `PublishAsync` holds `RLock` for the channel send; `Close` holds `Lock` for the `closed.Store + close(chan)` sequence.
- `Once` semantics are safe under concurrent `Publish`: an `atomic.Bool` per subscription (CAS steal pattern) ensures the handler fires at most once across all goroutines.

---

## Internal dispatch buckets

| Bucket | Pattern kind | Lookup cost | Allocs |
|---|---|---|---|
| `exact` | `user.created` | O(1) map lookup | 0 |
| `prefix` | `user.*` | O(1) via `lastDot` | 0 |
| `global` | `*` or `>` | O(1) iterate | 0 |
| `complex` | `user.>`, `tenant.*.user.>`, … | O(patterns) scan | 1 per match call (topic split) |
| `queueGroups` | exact only | O(1) map lookup | 0 |

---

## Future adapter compatibility

The `Broker` interface is the only public contract. Swap the implementation without changing any application code:

```go
// application code depends only on broker.Broker
var bus broker.Broker = broker.New()

// later, swap to a distributed broker
var bus broker.Broker = rbroker.New(redisClient, broker.Config{...})
```

The `matcher` package (`github.com/dangduoc08/ginject/matcher`) has **zero broker dependencies** and can be imported directly by any adapter that needs the same pattern-classification logic.
