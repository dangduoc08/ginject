# Broker

*Production-grade in-memory pub/sub event broker for the Ginject framework, with support for exact topics, prefix wildcards, global subscriptions, complex patterns, and queue groups.*

---

## Table of Contents

- [Broker](#broker)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [Constructors](#constructors)
    - [NewBroker](#newbroker)
    - [NewWithConfig](#newwithconfig)
  - [`Config` Parameters](#config-parameters)
    - [RecoverPanics](#recoverpanics)
    - [OnPanic](#onpanic)
    - [AsyncWorkers](#asyncworkers)
    - [AsyncQueueSize](#asyncqueuesize)
    - [BeforePublish](#beforepublish)
    - [AfterPublish](#afterpublish)
    - [BeforeDispatch](#beforedispatch)
    - [AfterDispatch](#afterdispatch)
  - [Types](#types)
    - [Message](#message)
    - [MessageHandler](#messagehandler)
    - [Subscription](#subscription)
    - [Stats](#stats)
  - [Broker Interface Methods](#broker-interface-methods)
    - [Publish](#publish)
    - [PublishAsync](#publishasync)
    - [Subscribe](#subscribe)
    - [Once](#once)
    - [SubscribeQueue](#subscribequeue)
    - [Unsubscribe](#unsubscribe)
    - [Off](#off)
    - [ListenerCount](#listenercount)
    - [Topics](#topics)
    - [Clear](#clear)
    - [Close](#close)
    - [Stats](#stats-method)
  - [Sentinel Errors](#sentinel-errors)
  - [Topic Patterns](#topic-patterns)
  - [Delivery Patterns](#delivery-patterns)
  - [Concurrency Model](#concurrency-model)
  - [Benchmarks](#benchmarks)

---

## Key Features

- **Exact, prefix, and complex wildcard patterns** — `user.created`, `user.*`, `user.>`, `tenant.*.user.>`
- **Multiple delivery modes** — fan-out, point-to-point, queue (competing consumer), broadcast
- **One-shot subscriptions** — `Once()` guarantees at-most-once delivery even under concurrent publish
- **Panic recovery** — configurable recovery; handlers that panic don't affect other handlers
- **Asynchronous publish** — bounded worker pool with backpressure (returns error on queue full, never blocks)
- **Observability hooks** — before/after publish and dispatch hooks for metrics and tracing
- **Lock-free atomic stats** — thread-safe counters without mutex contention
- **O(1) exact and prefix matching** — complex patterns use O(n) scan only when subscribed

---

## Usage

```go
package main

import (
	"fmt"

	"github.com/dangduoc08/ginject/broker"
)

func main() {
	// Create broker with safe defaults
	b := broker.NewBroker()
	defer b.Close()

	// Subscribe to an exact topic
	sub, err := b.Subscribe("user.created", func(m *broker.Message) {
		fmt.Printf("new user: %v\n", m.Payload)
	})
	if err != nil {
		panic(err)
	}

	// Publish to the topic
	err = b.Publish("user.created", map[string]string{
		"id":   "123",
		"name": "alice",
	})
	if err != nil {
		panic(err)
	}

	// Unsubscribe when done
	sub.Unsubscribe()
}
```

---

## Constructors

### NewBroker

Creates a broker with safe defaults: panic recovery enabled, async workers = `runtime.GOMAXPROCS(0)`, queue size = workers × 64.

```go
func NewBroker() Broker
```

**Returns**

- 1st value: `Broker`

- Description: A new in-memory broker instance

**Usage**

```go
b := broker.NewBroker()
defer b.Close()
```

---

### NewWithConfig

Creates a broker with custom configuration.

```go
func NewWithConfig(cfg Config) Broker
```

**Parameters**

- 1st parameter: `Config`

- Description: Configuration struct (see below)

**Returns**

- 1st value: `Broker`

- Description: A new broker configured as specified

**Usage**

```go
b := broker.NewWithConfig(broker.Config{
	RecoverPanics:  true,
	AsyncWorkers:   16,
	AsyncQueueSize: 2048,
	OnPanic: func(msg *broker.Message, r any) {
		log.Error("handler panic", "topic", msg.Topic, "error", r)
	},
})
defer b.Close()
```

---

## `Config` Parameters

### RecoverPanics

Type: `bool`

Default: `true` (via `NewBroker()`); `false` (via `NewWithConfig(Config{})`)

Required: `false`

When `true`, panics inside message handlers are caught and passed to `OnPanic` (if set), allowing other handlers to execute. When `false`, a panicking handler will propagate the panic.

```go
broker.NewWithConfig(broker.Config{
	RecoverPanics: true,
})
```

---

### OnPanic

Type: `func(*Message, any)`

Default: `nil`

Required: `false`

Callback invoked when a handler panics and `RecoverPanics` is `true`. The callback receives the message and the panic value. If `OnPanic` itself panics, the secondary panic is silently discarded.

```go
broker.NewWithConfig(broker.Config{
	RecoverPanics: true,
	OnPanic: func(msg *broker.Message, r any) {
		fmt.Printf("handler panic on %s: %v\n", msg.Topic, r)
	},
})
```

---

### AsyncWorkers

Type: `int`

Default: `0`

Required: `false`

Number of background goroutines for `PublishAsync`. Set to `0` to disable async publish (calls to `PublishAsync` return `ErrNoAsyncWorkers`). Typical value: `runtime.GOMAXPROCS(0)`.

```go
broker.NewWithConfig(broker.Config{
	AsyncWorkers: 8,
})
```

---

### AsyncQueueSize

Type: `int`

Default: `0` (defaults to `AsyncWorkers * 64` if `AsyncWorkers > 0`)

Required: `false`

Capacity of the async job queue. When the queue is full, `PublishAsync` returns `ErrAsyncQueueFull` immediately without blocking.

```go
broker.NewWithConfig(broker.Config{
	AsyncWorkers:   8,
	AsyncQueueSize: 512,
})
```

---

### BeforePublish

Type: `func(topic string, payload any)`

Default: `nil`

Required: `false`

Observability hook called before a message is published. If the hook panics, the panic is silently discarded and publish continues.

```go
broker.NewWithConfig(broker.Config{
	BeforePublish: func(topic string, _ any) {
		fmt.Println("publishing to", topic)
	},
})
```

---

### AfterPublish

Type: `func(topic string, payload any, err error)`

Default: `nil`

Required: `false`

Observability hook called after a message is published. Receives the topic, payload, and any error. Panics in the hook are silently discarded.

```go
broker.NewWithConfig(broker.Config{
	AfterPublish: func(topic string, _ any, err error) {
		if err == nil {
			metrics.Inc("broker.publish.success", topic)
		}
	},
})
```

---

### BeforeDispatch

Type: `func(msg *Message, handler int)`

Default: `nil`

Required: `false`

Observability hook called before each handler is invoked. `handler` is the index of the handler in the delivery list (0-based). Panics are silently discarded.

```go
broker.NewWithConfig(broker.Config{
	BeforeDispatch: func(msg *broker.Message, idx int) {
		span.AddEvent("dispatch", msg.Topic, idx)
	},
})
```

---

### AfterDispatch

Type: `func(msg *Message, handler int)`

Default: `nil`

Required: `false`

Observability hook called after each handler completes. Panics are silently discarded.

```go
broker.NewWithConfig(broker.Config{
	AfterDispatch: func(msg *broker.Message, idx int) {
		span.End()
	},
})
```

---

## Types

### Message

```go
type Message struct {
	ID        string         // UUID v4, generated at Publish time
	Topic     string         // exact topic string that was published
	Payload   any
	Timestamp time.Time
	Metadata  map[string]any // caller-managed; nil unless populated externally
}
```

**Fields**

- `ID` — Type: `string` — UUID v4 generated when the message is published; unique per publish call
- `Topic` — Type: `string` — The exact topic string passed to `Publish()`; does not include wildcard syntax
- `Payload` — Type: `any` — The data passed to `Publish()`
- `Timestamp` — Type: `time.Time` — Time when the message was created; set by `Publish()`
- `Metadata` — Type: `map[string]any` — Optional metadata; callers can populate this before subscribing; handlers can read or modify it

---

### MessageHandler

```go
type MessageHandler func(*Message)
```

A function that receives a message and processes it. Passed to `Subscribe()`, `Once()`, and `SubscribeQueue()`. If the broker is configured with `RecoverPanics: true`, panics in the handler are caught and passed to `OnPanic`.

---

### Subscription

```go
type Subscription interface {
	ID() string
	Topic() string
	Unsubscribe() error
}
```

Returned by `Subscribe()`, `Once()`, and `SubscribeQueue()`. Use it to unsubscribe or query subscription details.

**Methods**

- `ID() string` — Returns the unique subscription ID (UUID v4)
- `Topic() string` — Returns the pattern string passed to `Subscribe()`; for example, `"user.*"` or `"*"`
- `Unsubscribe() error` — Removes this subscription; subsequent publishes do not invoke its handler. Returns `ErrClosed` if the broker is closed; otherwise returns `nil`

---

### Stats

```go
type Stats struct {
	Topics          int
	Subscribers     int
	MessagesSent    uint64
	MessagesDropped uint64
	PublishCalls    uint64
}
```

Snapshot of broker state. Counters are `atomic.Uint64`, safe to read concurrently.

**Fields**

- `Topics` — Type: `int` — Number of active topics (patterns with at least one subscriber)
- `Subscribers` — Type: `int` — Total number of active subscriptions (including queue groups)
- `MessagesSent` — Type: `uint64` — Total messages delivered to all handlers (cumulative)
- `MessagesDropped` — Type: `uint64` — Total messages dropped due to `PublishAsync` queue overflow
- `PublishCalls` — Type: `uint64` — Total calls to `Publish()` or `PublishAsync()` (includes failed calls)

---

## Broker Interface Methods

### Publish

Publishes a message to a topic synchronously. Blocks until all matching handlers have executed.

```go
func (b *MemoryBroker) Publish(topic string, payload any) error
```

**Parameters**

- 1st parameter: `string` — The topic to publish to
- 2nd parameter: `any` — The payload; can be any Go value (struct, string, []byte, etc.)

**Returns**

- 1st value: `error` — `ErrEmptyTopic` if topic is `""`, `ErrClosed` if broker is closed; `nil` on success

**Rules**

- Blocks until all handlers (exact, prefix, wildcard, and queue groups) have been invoked
- If a handler panics and `RecoverPanics` is `true`, the panic is caught; other handlers still execute
- Once handlers fire exactly once (then remove themselves); they do not fire on subsequent publishes
- Empty topic returns `ErrEmptyTopic` immediately without calling handlers
- Closed broker returns `ErrClosed` immediately

**Usage**

```go
err := b.Publish("user.created", map[string]string{
	"id":   "123",
	"name": "alice",
})
if err != nil {
	log.Fatal(err)
}
```

---

### PublishAsync

Publishes a message asynchronously via a worker pool. Returns immediately; delivery happens in the background.

```go
func (b *MemoryBroker) PublishAsync(topic string, payload any) error
```

**Parameters**

- 1st parameter: `string` — The topic to publish to
- 2nd parameter: `any` — The payload

**Returns**

- 1st value: `error` — `ErrEmptyTopic` if topic is `""`, `ErrClosed` if broker is closed, `ErrNoAsyncWorkers` if `AsyncWorkers == 0`, `ErrAsyncQueueFull` if queue is full; `nil` on success

**Rules**

- Returns immediately; delivery is fire-and-forget
- If `AsyncWorkers == 0`, returns `ErrNoAsyncWorkers`
- If the async queue is full, increments `MessagesDropped` and returns `ErrAsyncQueueFull` (never blocks)
- Handlers fire in the background; caller has no visibility into delivery or errors
- Use for non-critical operations: metrics, audit logs, notifications

**Usage**

```go
err := b.PublishAsync("metrics.collected", stats)
if errors.Is(err, broker.ErrAsyncQueueFull) {
	log.Warn("async queue full; metric dropped")
}
```

---

### Subscribe

Subscribes a handler to a topic pattern. Handler is called every time a matching message is published.

```go
func (b *MemoryBroker) Subscribe(topic string, handler MessageHandler) (Subscription, error)
```

**Parameters**

- 1st parameter: `string` — The topic pattern (e.g., `"user.created"`, `"order.*"`, `"*"`)
- 2nd parameter: `MessageHandler` — The handler function

**Returns**

- 1st value: `Subscription` — Use this to query or unsubscribe
- 2nd value: `error` — `ErrEmptyTopic` if topic is `""`, `ErrNilHandler` if handler is `nil`, `ErrClosed` if broker is closed; `nil` on success

**Rules**

- Handler is invoked synchronously on each `Publish()` to a matching topic
- Multiple subscriptions to the same topic pattern are independent; all are invoked
- Handler is kept alive until `Unsubscribe()` is called
- Nil handler returns `ErrNilHandler`

**Usage**

```go
sub, err := b.Subscribe("user.created", func(m *broker.Message) {
	fmt.Printf("user %v was created\n", m.Payload)
})
if err != nil {
	panic(err)
}
defer sub.Unsubscribe()
```

---

### Once

Subscribes a handler to a topic pattern, but guarantees the handler fires **at most once** across all publishes, even under concurrent publish calls.

```go
func (b *MemoryBroker) Once(topic string, handler MessageHandler) (Subscription, error)
```

**Parameters**

- 1st parameter: `string` — The topic pattern
- 2nd parameter: `MessageHandler` — The handler function

**Returns**

- 1st value: `Subscription` — Use this to query or manually unsubscribe before the handler fires
- 2nd value: `error` — `ErrEmptyTopic`, `ErrNilHandler`, `ErrClosed`; `nil` on success

**Rules**

- Handler fires exactly once (or zero times if never published or manually unsubscribed)
- Thread-safe: if multiple goroutines publish concurrently to a topic with a `Once` subscriber, the handler is guaranteed to fire exactly once (guarded by `atomic.Bool` and CAS)
- After the handler fires, the subscription is automatically removed
- Does not fire on subsequent publishes to the same or similar topics

**Usage**

```go
b.Once("app.started", func(m *broker.Message) {
	fmt.Println("startup hook fired")
})

b.Publish("app.started", nil)     // fires
b.Publish("app.started", nil)     // no-op; subscription was removed
```

---

### SubscribeQueue

Subscribes a handler as part of a named queue group. On each publish, exactly one handler in the group is invoked (round-robin).

```go
func (b *MemoryBroker) SubscribeQueue(topic, group string, handler MessageHandler) (Subscription, error)
```

**Parameters**

- 1st parameter: `string` — An exact topic (no wildcards); e.g., `"task.process"`
- 2nd parameter: `string` — The group name; multiple handlers with the same group name form one queue
- 3rd parameter: `MessageHandler` — The handler function

**Returns**

- 1st value: `Subscription` — Use this to query or unsubscribe
- 2nd value: `error` — `ErrEmptyTopic`, `ErrEmptyGroup`, `ErrNilHandler`, `ErrWildcardInQueue` (if topic contains `*` or `>`), `ErrClosed`; `nil` on success

**Rules**

- Requires an exact topic (no `*` or `>` wildcards); wildcard topics return `ErrWildcardInQueue`
- Handlers in the same group round-robin: each publish invokes one handler, then the next handler, etc.
- Multiple groups on the same topic are independent; each group receives one delivery per publish
- Fan-out (`Subscribe`) and queue subscriptions coexist on the same topic
- Empty group name returns `ErrEmptyGroup`

**Usage**

```go
// Three workers competing for jobs
for i := 0; i < 3; i++ {
	i := i
	b.SubscribeQueue("task.process", "workers", func(m *broker.Message) {
		fmt.Printf("worker %d: %v\n", i, m.Payload)
	})
}

b.Publish("task.process", job1) // → worker 0
b.Publish("task.process", job2) // → worker 1
b.Publish("task.process", job3) // → worker 2
b.Publish("task.process", job4) // → worker 0 (round-robin)
```

---

### Unsubscribe

Removes a subscription so the handler is no longer invoked.

```go
func (b *MemoryBroker) Unsubscribe(sub Subscription) error
```

**Parameters**

- 1st parameter: `Subscription` — The subscription to remove (or `nil`, which is a no-op)

**Returns**

- 1st value: `error` — `ErrClosed` if broker is closed; otherwise `nil`

**Rules**

- Nil `Subscription` is safe; returns `nil` without error
- Handler is removed synchronously; subsequent publishes do not invoke it
- Safe to call multiple times on the same subscription (idempotent)
- Works for all subscription types: `Subscribe`, `Once`, `SubscribeQueue`

**Usage**

```go
sub, _ := b.Subscribe("event", handler)
// ... later ...
b.Unsubscribe(sub) // or sub.Unsubscribe()
```

---

### Off

Removes all subscriptions matching a topic pattern (exact pattern match only, no wildcard expansion).

```go
func (b *MemoryBroker) Off(topic string) error
```

**Parameters**

- 1st parameter: `string` — The exact pattern to clear (e.g., `"user.created"`, `"order.*"`, `"*"`)

**Returns**

- 1st value: `error` — `ErrClosed` if broker is closed; otherwise `nil`

**Rules**

- Removes all subscriptions registered with this exact pattern (not topics that match the pattern)
- `Off("*")` removes all global wildcard subscriptions
- `Off("order.*")` removes all prefix-wildcard subscriptions for `order.*`
- `Off("user.created")` removes exact subscriptions for `user.created` and queue groups for `user.created`
- Does not affect subscriptions to other patterns (e.g., `Off("user.*")` does not remove `Subscribe("user.created")`)

**Usage**

```go
b.Off("user.created")  // remove all handlers for this exact topic
b.Off("order.*")       // remove all handlers for the order.* pattern
b.Off("*")             // remove all global handlers
```

---

### ListenerCount

Returns the number of active handlers for a topic pattern.

```go
func (b *MemoryBroker) ListenerCount(topic string) int
```

**Parameters**

- 1st parameter: `string` — The topic pattern

**Returns**

- 1st value: `int` — Count of handlers (including queue group members)

**Rules**

- Returns count for the exact pattern passed; does not expand wildcards
- Counts include exact subscriptions, queue group members, and wildcard subscriptions for that pattern
- Returns 0 if no handlers are registered for this pattern

**Usage**

```go
n := b.ListenerCount("user.created")
if n == 0 {
	fmt.Println("no handlers for user.created")
}
```

---

### Topics

Returns a list of all active topic patterns (topics with at least one handler).

```go
func (b *MemoryBroker) Topics() []string
```

**Returns**

- 1st value: `[]string` — Slice of topic patterns

**Rules**

- Returns only patterns with at least one active subscription
- Empty broker returns empty slice
- Topics are deduplicated (each pattern appears at most once)
- Order is undefined; caller must sort if order matters

**Usage**

```go
topics := b.Topics()
for _, t := range topics {
	fmt.Printf("topic: %s, listeners: %d\n", t, b.ListenerCount(t))
}
```

---

### Clear

Removes all subscriptions without closing the broker.

```go
func (b *MemoryBroker) Clear() error
```

**Returns**

- 1st value: `error` — `ErrClosed` if broker is closed; otherwise `nil`

**Rules**

- Removes all subscriptions (exact, prefix, global, complex, queue groups)
- Broker remains open and can accept new subscriptions
- Stats counters are not reset (MessagesSent, PublishCalls, etc. persist)

**Usage**

```go
b.Clear() // remove all handlers
```

---

### Close

Closes the broker, preventing any further subscriptions or publishes.

```go
func (b *MemoryBroker) Close() error
```

**Returns**

- 1st value: `error` — Always `nil` (no error conditions)

**Rules**

- All subsequent calls to `Publish`, `PublishAsync`, `Subscribe`, `Once`, `SubscribeQueue`, `Off`, `Clear` return `ErrClosed`
- If `AsyncWorkers > 0`, waits for the async worker pool to drain before returning
- Clears all subscription maps after closing
- Safe to call multiple times (idempotent)
- Safe to call from any goroutine

**Usage**

```go
defer b.Close()
// or explicitly
if err := b.Close(); err != nil {
	log.Error(err)
}
```

---

### Stats (method)

Returns a snapshot of broker statistics.

```go
func (b *MemoryBroker) Stats() Stats
```

**Returns**

- 1st value: `Stats` — A snapshot of current broker state

**Rules**

- Counters are `atomic.Uint64`; safe to read concurrently without locking
- Snapshot is point-in-time; counters may change immediately after the call returns
- `Topics` and `Subscribers` counts are current; `MessagesSent`, `MessagesDropped`, `PublishCalls` are cumulative

**Usage**

```go
stats := b.Stats()
fmt.Printf("topics=%d subscribers=%d sent=%d\n", stats.Topics, stats.Subscribers, stats.MessagesSent)
```

---

## Sentinel Errors

The broker defines a set of sentinel error values for validation and state errors:

```go
var (
	ErrClosed          = errors.New("broker: broker is closed")
	ErrNilHandler      = errors.New("broker: handler must not be nil")
	ErrEmptyTopic      = errors.New("broker: topic must not be empty")
	ErrEmptyGroup      = errors.New("broker: group must not be empty")
	ErrAsyncQueueFull  = errors.New("broker: async queue full")
	ErrNoAsyncWorkers  = errors.New("broker: PublishAsync requires AsyncWorkers > 0")
	ErrWildcardInQueue = errors.New("broker: SubscribeQueue requires an exact topic")
)
```

Use `errors.Is()` to check for specific errors:

```go
err := b.PublishAsync("topic", data)
if errors.Is(err, broker.ErrAsyncQueueFull) {
	// queue is full; metric dropped
}
```

---

## Topic Patterns

Patterns are parsed once at subscribe time by the `matcher` package. Publish uses different lookup strategies based on pattern type.

| Pattern | Kind | Lookup | Example matches | Example non-matches |
|---|---|---|---|---|
| `user.created` | Exact | O(1) map | `user.created` | `user.updated` |
| `*` | Global | O(1) | every topic | — |
| `>` | Global | O(1) | every topic | — |
| `user.*` | Suffix | O(1) | `user.created`, `user.deleted` | `user.profile.updated` |
| `user.>` | Complex | O(n) | `user.created`, `user.profile.updated` | `user` (no dot) |
| `tenant.*.user.>` | Complex | O(n) | `tenant.1.user.created`, `tenant.1.user.a.b` | `tenant.1.user` |
| `*.created` | Complex | O(n) | `user.created`, `order.created` | `a.b.created` |

**Notes:**

- `*` and `>` are equivalent (both match all topics); `*` is supported for backward compatibility
- `user.*` matches only one level below `user.` (i.e., exactly `user.<something>`)
- `user.>` matches `user.` and any number of additional levels (greedy suffix)
- Complex patterns require `O(patterns)` scan; use exact or suffix when possible

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
b.Subscribe("conn."+connID, func(m *broker.Message) {
	ws.WriteJSON(m.Payload)
})

b.Publish("conn.abc123", payload) // to one client
```

### 3. Room / Group — Shared topic

```go
b.Subscribe("room.42", clientA)
b.Subscribe("room.42", clientB)

b.Publish("room.42", chatMsg) // both clients receive
```

### 4. Queue / Competing Consumer — One handler per group

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
// billing receives once AND inventory receives once
```

### 5. Broadcast — Global wildcard

```go
b.Subscribe("*", func(m *broker.Message) {
	fmt.Printf("[audit] %s %v\n", m.Topic, m.Payload)
})

b.Publish("anything.at.all", data) // audit handler fires
```

---

## Concurrency Model

- **Single `sync.RWMutex`** protects all subscription maps (exact, prefix, global, complex, queue groups). No nested locks.
- **Publish** acquires `RLock` only long enough to snapshot handler references, then releases before invoking handlers. This prevents deadlock if a handler re-enters the broker.
- **PublishAsync vs Close race** is eliminated via a separate `closeMu sync.RWMutex`: `PublishAsync` holds `RLock` for the channel send; `Close` holds `Lock` to set closed flag and close the channel.
- **Once semantics** are safe under concurrent `Publish`: atomic `Bool` and CAS (compare-and-swap) ensure the handler fires at most once across all goroutines.
- **Stats counters** are `atomic.Uint64`; no mutex contention on reads.

---

## Benchmarks

Benchmarks were run on an Intel Core i7-9750H CPU @ 2.60 GHz with `go test -bench=. -benchmem`.

Results are machine-dependent and were captured at documentation generation time (July 30, 2026).

| Benchmark | Operations | Time per op | Allocs | Bytes |
|---|---|---|---|---|
| `BenchmarkPublish` (1000 subscribers, exact) | 43,189 | 25,641 ns/op | 10 | 17,608 B/op |
| `BenchmarkPublishWildcard` (100 wildcard subscribers) | 389,816 | 3,092 ns/op | 10 | 2,304 B/op |
| `BenchmarkPublishMixed` (10 exact, 10 prefix, 10 global) | 770,823 | 1,515 ns/op | 5 | 583 B/op |
| `BenchmarkSubscribe` (subscribe + unsubscribe loop) | 1,467,124 | 875.8 ns/op | 5 | 480 B/op |
| `BenchmarkPublishParallel` (concurrent publish, 10 subscribers) | 952,686 | 1,356 ns/op | 4 | 328 B/op |
| `BenchmarkPublishManyTopics` (1000 topics, round-robin) | 1,481,018 | 812.5 ns/op | 4 | 157 B/op |

**Key observations:**

- Exact-match publish scales linearly with subscriber count (25µs for 1000 subscribers)
- Wildcard subscriptions incur minimal overhead (~3µs)
- Mixed patterns (exact + prefix + global) are fast due to efficient bucket lookup
- Subscribe and unsubscribe are sub-microsecond operations
- Concurrent publish scales well across CPU cores with minimal lock contention
