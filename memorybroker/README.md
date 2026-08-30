# memorybroker

*Lightweight in-process topic-based pub/sub broker for the Ginject framework — exact topics, prefix wildcards, global subscriptions, and complex patterns.*

---

## Table of Contents

- [memorybroker](#memorybroker)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [Constructor](#constructor)
    - [NewMemoryBroker](#newmemorybroker)
  - [Types](#types)
    - [Message](#message)
    - [MessageHandler](#messagehandler)
    - [Subscription](#subscription)
  - [Broker Interface Methods](#broker-interface-methods)
    - [Publish](#publish)
    - [PublishAsync](#publishasync)
    - [Subscribe](#subscribe)
    - [Unsubscribe](#unsubscribe)
    - [Close](#close)
  - [Sentinel Errors](#sentinel-errors)
  - [Topic Patterns](#topic-patterns)
  - [Delivery Patterns](#delivery-patterns)
  - [Concurrency Model](#concurrency-model)
  - [Benchmarks](#benchmarks)

---

## Key Features

- **Exact, prefix, and complex wildcard patterns** — `user.created`, `user.*`, `tenant.*.user.created`, `tenant.*.user.*`
- **Fan-out delivery** — every matching subscriber receives every publish
- **Fire-and-forget async publish** — `PublishAsync` delivers on its own goroutine and returns immediately
- **Panic recovery** — a panicking handler is recovered and does not affect other handlers or the broker
- **O(1) exact and prefix matching** — complex patterns use O(n) scan only when subscribed

This is an in-process message bus, not a distributed broker: no persistence, no durable queues, no consumer groups, no clustering, no bounded async worker pool. Those concerns belong to an external system (NATS, Kafka, RabbitMQ) if the application ever needs them. The public API is deliberately kept to the methods Ginject's own request pipeline and WebSocket layer actually use — see [Broker Interface Methods](#broker-interface-methods).

---

## Usage

```go
package main

import (
	"fmt"

	"github.com/dangduoc08/ginject/memorybroker"
)

func main() {
	b := memorybroker.NewMemoryBroker()
	defer b.Close()

	sub, err := b.Subscribe("user.created", func(m *memorybroker.Message) {
		fmt.Printf("new user: %v\n", m.Payload)
	})
	if err != nil {
		panic(err)
	}

	err = b.Publish("user.created", map[string]string{
		"id":   "123",
		"name": "alice",
	})
	if err != nil {
		panic(err)
	}

	sub.Unsubscribe()
}
```

---

## Constructor

### NewMemoryBroker

```go
func NewMemoryBroker() Broker
```

Creates a broker ready to use, with panic recovery on every handler dispatch. There is no configuration struct — the broker has no tunable knobs.

**Usage**

```go
b := memorybroker.NewMemoryBroker()
defer b.Close()
```

---

## Types

### Message

```go
type Message struct {
	Topic     string
	Payload   any
	Timestamp time.Time
}
```

**Fields**

- `Topic` — Type: `string` — The exact topic string passed to `Publish()`; does not include wildcard syntax
- `Payload` — Type: `any` — The data passed to `Publish()`
- `Timestamp` — Type: `time.Time` — Time when the message was created; set by `Publish()`

---

### MessageHandler

```go
type MessageHandler func(*Message)
```

A function that receives a message and processes it. Passed to `Subscribe()`. Panics in the handler are always recovered and do not stop delivery to other handlers.

---

### Subscription

```go
type Subscription interface {
	ID() string
	Topic() string
	Unsubscribe() error
}
```

Returned by `Subscribe()`. Use it to unsubscribe or query subscription details.

**Methods**

- `ID() string` — Returns the unique subscription ID (UUID v4)
- `Topic() string` — Returns the pattern string passed at subscribe time; for example, `"user.*"` or `"*"`
- `Unsubscribe() error` — Removes this subscription; subsequent publishes do not invoke its handler. Returns `ErrClosed` if the broker is closed; otherwise returns `nil`

---

## Broker Interface Methods

### Publish

Publishes a message to a topic synchronously. Blocks until all matching handlers have executed.

```go
func (b *MemoryBroker) Publish(topic string, payload any) error
```

**Returns**: `ErrEmptyTopic` if topic is `""`, `ErrClosed` if broker is closed; `nil` on success.

**Rules**

- Blocks until all handlers (exact, prefix, wildcard, complex) have been invoked
- A panicking handler is recovered; other handlers still execute

**Usage**

```go
err := b.Publish("user.created", map[string]string{"id": "123", "name": "alice"})
```

---

### PublishAsync

Publishes a message on its own goroutine and returns immediately.

```go
func (b *MemoryBroker) PublishAsync(topic string, payload any) error
```

**Returns**: `ErrEmptyTopic` if topic is `""`, `ErrClosed` if broker is closed; `nil` on success.

**Rules**

- Returns immediately; delivery is fire-and-forget on a dedicated goroutine per call
- Handlers fire in the background; caller has no visibility into delivery or errors
- `Close()` waits for every in-flight `PublishAsync` goroutine to finish before returning
- Use for non-critical operations: metrics, audit logs, notifications

**Usage**

```go
err := b.PublishAsync("metrics.collected", stats)
```

---

### Subscribe

Subscribes a handler to a topic pattern. Handler is called every time a matching message is published.

```go
func (b *MemoryBroker) Subscribe(topic string, handler MessageHandler) (Subscription, error)
```

**Returns**: `Subscription`, and `error` (`ErrEmptyTopic`, `ErrNilHandler`, `ErrClosed`; `nil` on success).

**Rules**

- Handler is invoked synchronously on each `Publish()` to a matching topic
- Multiple subscriptions to the same topic pattern are independent; all are invoked
- Handler is kept alive until `Unsubscribe()` is called

**Usage**

```go
sub, err := b.Subscribe("user.created", func(m *memorybroker.Message) {
	fmt.Printf("user %v was created\n", m.Payload)
})
defer sub.Unsubscribe()
```

---

### Unsubscribe

Removes a subscription so the handler is no longer invoked.

```go
func (b *MemoryBroker) Unsubscribe(sub Subscription) error
```

**Rules**

- Nil `Subscription` is always a no-op that returns `nil` — unconditionally, even after `Close`; this is the one exception to Close's "all subsequent calls return `ErrClosed`" rule, since there is nothing to unsubscribe
- Handler is removed synchronously; subsequent publishes do not invoke it
- Safe to call multiple times on the same subscription (idempotent)
- A `Subscription` obtained from a different `MemoryBroker` instance is rejected with `ErrForeignSubscription` and has no effect on either broker

---

### Close

Closes the broker, preventing any further subscriptions or publishes.

```go
func (b *MemoryBroker) Close() error
```

**Rules**

- All subsequent calls to `Publish`, `PublishAsync`, `Subscribe` return `ErrClosed`; `Unsubscribe` does too, except a `nil` `Subscription`, which is always `nil` (see Unsubscribe above)
- Waits for every in-flight `PublishAsync` goroutine to finish before returning — it does **not** wait for a synchronous `Publish` call running on another goroutine, since `Publish` is not tracked by the same `sync.WaitGroup`
- Clears all subscription maps after closing
- Safe to call multiple times (idempotent) and from any goroutine, **except** from within a handler that was itself dispatched by `PublishAsync`: that goroutine's own `wg.Done()` only runs after the handler returns, so a synchronous `Close()` call from inside it blocks forever waiting on its own completion. Calling `Close()` from within a handler dispatched by the synchronous `Publish` is safe (`Publish` isn't `wg`-tracked); if you need to close from an async handler, do it asynchronously (e.g. `go b.Close()`) or via a signal to another goroutine

---

## Sentinel Errors

```go
var (
	ErrClosed              = errors.New("memorybroker: broker is closed")
	ErrNilHandler          = errors.New("memorybroker: handler must not be nil")
	ErrEmptyTopic          = errors.New("memorybroker: topic must not be empty")
	ErrForeignSubscription = errors.New("memorybroker: subscription does not belong to this broker")
)
```

Use `errors.Is()` to check for specific errors.

---

## Topic Patterns

Patterns are parsed once at subscribe time by the `pattern` package. Publish uses different lookup strategies based on pattern type.

| Pattern | Kind | Lookup | Example matches | Example non-matches |
|---|---|---|---|---|
| `user.created` | Exact | O(1) map | `user.created` | `user.updated` |
| `*` | Global | O(1) | every topic | — |
| `user.*` | Suffix | O(1) | `user.created`, `user.profile.updated` (any depth) | `user` (no dot) |
| `tenant.*.user.created` | Complex | O(n) | `tenant.1.user.created`, `tenant.abc.user.created` | `tenant.1.user.updated` |
| `tenant.*.user.*` | Complex | O(n) | `tenant.1.user.created`, `tenant.abc.user.profile.updated` | `tenant.1.admin.created` |
| `*.created` | Complex | O(n) | `user.created`, `order.created` | `a.b.created` |

**Notes:**

- `*` is the **only** wildcard token. `>` is not special-cased anywhere in the `pattern` package — a pattern containing `>` (e.g. `user.>`) is parsed as a literal segment and will only match that exact literal topic string, never as a wildcard. Do not use `>`; this table previously claimed `>` was a global-wildcard alias, which was never true of this implementation — verified empirically, not just by reading the code
- `user.*` matches any depth below `user.` (greedy suffix), not just one level
- Complex patterns require an `O(patterns)` scan; use exact or suffix when possible

---

## Delivery Patterns

### 1. Fan-Out — All subscribers receive

```go
b.Subscribe("order.created", notifyEmail)
b.Subscribe("order.created", notifySlack)
b.Subscribe("order.created", updateInventory)

b.Publish("order.created", order) // all three handlers fire
```

### 2. Direct / Point-to-Point — Unique topic per connection

```go
b.Subscribe("conn."+connID, func(m *memorybroker.Message) {
	ws.WriteJSON(m.Payload)
})

b.Publish("conn.abc123", payload) // to one client
```

This is how the WebSocket layer wires topics to connections: `core/ws_connmgr.go` calls `Subscribe`/`Unsubscribe` per connection and topic, and `core/read_loop.go` calls `Publish` when a client publishes into a topic it's subscribed to — fan-out to every other subscriber of that topic happens for free.

### 3. Room / Group — Shared topic

```go
b.Subscribe("room.42", clientA)
b.Subscribe("room.42", clientB)

b.Publish("room.42", chatMsg) // both clients receive
```

### 4. Broadcast — Global wildcard

```go
b.Subscribe("*", func(m *memorybroker.Message) {
	fmt.Printf("[audit] %s %v\n", m.Topic, m.Payload)
})

b.Publish("anything.at.all", data) // audit handler fires
```

---

## Concurrency Model

- **Single `sync.RWMutex`** protects all subscription maps (exact, prefix, global, complex) — no nested locks, no lock-ordering to reason about.
- **Publish** acquires `RLock` only long enough to snapshot handler references, then releases before invoking handlers. This prevents deadlock if a handler re-enters the broker, and means user handler code never runs while a lock is held.
- **Subscribe, Unsubscribe, and PublishAsync vs Close** all check `closed` under the same lock they use to mutate state (`Subscribe`/`Unsubscribe` under `Lock`, `PublishAsync` under `RLock` while it registers with `sync.WaitGroup`), so the check-then-act is atomic with `Close`'s `CompareAndSwap`. `Close` takes `Lock` to flip the closed flag, then calls `wg.Wait()` outside the lock so it's guaranteed to observe every `PublishAsync` goroutine that started before it closed. This guarantees no subscription can be inserted, and no in-flight async publish can be missed, once `Close` has committed.
- **Panic recovery** wraps every handler call unconditionally — a panicking handler cannot crash the broker or skip other handlers.
- **Close** is idempotent: the closed flag is flipped via `CompareAndSwap`, so a second call returns immediately without waiting or re-clearing the maps.

---

## Benchmarks

Run with `go test -bench=. -benchmem ./memorybroker/...` on an Intel Core i7-9750H CPU @ 2.60 GHz.

| Benchmark | Time per op | Allocs | Bytes |
|---|---|---|---|
| `BenchmarkPublish` (1000 subscribers, exact) | 20,934 ns/op | 2 | 8,256 B/op |
| `BenchmarkPublishWildcard` (100 wildcard subscribers) | 1,803 ns/op | 2 | 960 B/op |
| `BenchmarkPublishMixed` (10 exact, 10 prefix, 10 global) | 923.9 ns/op | 3 | 544 B/op |
| `BenchmarkSubscribe` (subscribe + unsubscribe loop) | 824.8 ns/op | 5 | 450 B/op |
| `BenchmarkPublishParallel` (concurrent publish, 10 subscribers) | 268.4 ns/op | 2 | 227 B/op |
| `BenchmarkPublishManyTopics` (1000 topics, round-robin) | 425.2 ns/op | 3 | 204 B/op |
