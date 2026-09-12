# WebSocket System & Pub/Sub Broker

**Optimization**: Connection lifecycle state machine, pub/sub decision trees, concurrency guarantees.

**Broker Package**: `memorybroker` (in-memory pub/sub for WebSocket fanout, renamed from `broker` in v2.1)

## 0. Transport Limits (read first)

| `WSConfig` field | Default constant | Value | Effect |
|---|---|---|---|
| `MaxConnections` | `DefaultWSMaxConnections` | 10000 | `Register` returns **nil** past the cap or on duplicate `connID`; `handleRequest` closes the socket and returns |
| `MaxPayloadBytes` | `DefaultWSMaxPayloadBytes` | 1 MB | Set on `wsConn.MaxPayloadBytes`. `x/net` would otherwise default to 32 MB |
| `WriteTimeout` | `DefaultWSWriteTimeout` | 10 s | Deadline before every `Send`; a write error closes the conn to force teardown |
| `AllowedOrigins` | — | nil | When non-empty, `handshake` rejects unlisted `Origin` with `errWSOriginRejected`. Absent `Origin` is allowed |

`0` keeps the default for the first three. Full rationale and opt-outs: [security-limits-and-defaults.md](security-limits-and-defaults.md).

**`Register` may return nil** — every caller must nil-check. This is deliberate: refusing a duplicate `connID` rather than replacing it stops a client that can influence `connID` from evicting someone else's connection.

## 1. WebSocket Connection Lifecycle

### 1.1 State Machine

```
[INIT]
  ↓
[HANDSHAKE] — HTTP middleware chain runs (global + WS-specific)
  ├─ A middleware that does not call next() rejects the upgrade
  └─ Context set on the request survives into the connection
  ↓
[ACCEPTED] — websocket.Conn established
  ├─ connID generated (UUID)
  └─ Registered in WSConnMgr
  ↓
[RUNNING] — Messages flow
  ├─ readLoop() blocking on conn.Receive()
  ├─ writeLoop() draining send channel
  └─ Handlers process messages
  ↓
[CLOSED] — Client disconnects or error
  ├─ readLoop exits
  ├─ Unregister() closes done channel
  ├─ writeLoop exits
  └─ All subscriptions cleaned up
```

### 1.2 Connection Lifecycle Phases

**Phase 1: HTTP Upgrade**
```
Client sends: GET /ws HTTP/1.1 Upgrade: websocket
Server receives upgrade request
```

**Phase 2: Handshake**
```
app.ws.upgrade(w, r, websocket.Server{
    Handshake: func(wsCfg, r) error {
        // Origin allowlist, then the HTTP middleware chain:
        //   1. everything bound with app.BindGlobalMiddlewares
        //   2. everything passed to app.EnableWS(cfg, ...)
        // A middleware that does not call next() rejects the upgrade.
        // There is NO Guard and NO Interceptor at handshake time —
        // those run per subscribe/unsubscribe/publish instead.
        return nil  // Accept connection
    },
    Handler: websocket.Handler(app.ws.handleRequest),
})
```

Both success and failure emit a `StageComplete` trace event with
`Operation: "handshake"`, `Status: ok|rejected|failed` and the real HTTP
status code — including the case where the request never reaches the
handshake callback at all (a plain GET on the WS path).

**Phase 3: Connection Acceptance**
```
websocket.Conn accepted
connID := generateUUID()
WSConnMgr.Register(connID, conn)
    ├─ sendChan := make(chan WSPayload, 32)
    ├─ writeLoop() spawned
    └─ readLoop() spawned
```

**Phase 4: Message Exchange**
```
readLoop() — Per-connection goroutine
    ├─ Blocks on conn.Receive()
    ├─ Receives JSON message
    ├─ Parses message
    ├─ Matches to handler
    ├─ Invokes handler (same pipeline as HTTP)
    └─ Handler can memorybroker.Publish()

writeLoop() — Per-connection goroutine
    ├─ Drains sendChan
    ├─ Writes each message to conn
    └─ Non-blocking on send: TrySend()
```

**Phase 5: Cleanup**
```
Client closes connection OR error in readLoop
    ↓
readLoop exits
    ↓
Unregister(connID)
    ├─ Closes done channel
    └─ Triggers writeLoop exit
    ↓
writeLoop exits
    ↓
All memorybroker subscriptions cleaned up
    ↓
Connection resources freed
```

---

## 2. Message Flow

### 2.1 Inbound Message Processing

```
readLoop() receives raw message
    ↓
Unmarshal JSON → WSPayload{Type, Topic, ID, Message, ...}
    ↓
Update LastSeen timestamp (for dead connection detection)
    ↓
Pattern matching: Type
    ├─ TypeSubscribe → handleSubscribe        (Guards only, NO Interceptors)
    │   ├─ Validate topic (non-empty, <= 255 chars)
    │   ├─ Match handler for topic
    │   ├─ Run Guard chain (sees ConnID, Topic, Pattern, Operation)
    │   ├─ Register memorybroker subscription callback
    │   └─ reply TypeAck | TypeError with []WSTopicResult
    │
    ├─ TypePublish → handlePublish
    │   ├─ Validate topic
    │   ├─ Match handler for topic
    │   ├─ Verify subscription exists
    │   ├─ Guard → Interceptor → Pipe → Handler → ExceptionFilter
    │   ├─ reply TypeResponse (handler return value, carries Topic)
    │   ├─ memorybroker.Publish(topic, Message) for fanout
    │   └─ reply TypeAck | TypeError with []WSTopicResult
    │
    ├─ TypeUnsubscribe → handleUnsubscribe     (Guards only, NO Interceptors)
    │   ├─ Run Guard chain
    │   ├─ Unregister memorybroker callback
    │   └─ reply TypeAck | TypeError with []WSTopicResult
    │
    ├─ TypePing → reply(conn, TypePong, ID, "") ← echoes the ping id
    │
    ├─ TypePong → Record liveness (no response, LastSeen updated above)
    │
    └─ Other types → reply(conn, TypeError, ID, message) ← Error response
```

**ACK Protocol**:
- Subscribe, Unsubscribe and Publish all carry a request `ID` and are all acked
- The reply echoes that `ID` and carries `[]WSTopicResult` — one entry per
  requested topic, so a mixed batch reports each topic's outcome instead of
  aborting at the first failure
- `TypeAck` when every topic succeeded, `TypeError` when any did not
- When the pipeline already answered with an exception frame (Guard denial,
  handler panic) and that covers every requested topic, no second summary
  frame is sent

**Response vs Event** — two different frames, do not conflate them:
- `TypeResponse` carries a handler's return value back to the **publisher**,
  tagged with the request `ID` and the `Topic` it answers
- `TypeEvent` carries a broker fan-out to **subscribers**, tagged with the
  concrete `Topic` and the `Pattern` the connection subscribed with (so a
  `chat.*` subscriber can route a `chat.123` event)

### 2.2 Outbound Message Broadcasting

```
memorybroker.Publish("users.created", userData)
    ↓
Memorybroker looks up subscribers for "users.created"
    ↓
For each subscriber connection:
    ├─ Get subscription callback (fanout handler)
    ├─ Call callback(userData)
    └─ Callback calls conn.TrySend(message) ← NON-BLOCKING
    ↓
If send channel buffer full (SendBufferSize, default 32):
    ├─ TrySend() returns false and increments the connection's drop counter
    ├─ First drop logs WSSlowConsumer and marks the conn a slow consumer
    └─ Past MaxDroppedFrames the connection is evicted (WSSlowConsumerEvicted)
       rather than silently losing more frames
```

---

## 3. WebSocket Payload Structure

### 3.1 WSPayload (v2.0+)

```go
type WSPayload struct {
    Type    WSPayloadType  // see the constants below
    ID      string         // request/response correlation ID
    Topic   []string       // topic name(s) - array for batch operations
    Pattern string         // on TypeEvent: the subscription that delivered it
    Message any            // message data (marshals to/from JSON)
}

type WSTopicResult struct {
    Topic   string  // the requested topic
    Status  string  // "ok" | "rejected" | "failed"
    Code    int     // WS close-style code when not ok
    Error   string  // short error name
    Message string  // human-readable detail
}

type WSPayloadType string
const (
    TypeSubscribe   WSPayloadType = "subscribe"
    TypeUnsubscribe WSPayloadType = "unsubscribe"
    TypePublish     WSPayloadType = "publish"
    TypeEvent       WSPayloadType = "event"      // broker fan-out to a subscriber
    TypeResponse    WSPayloadType = "response"   // a handler's return value to the publisher
    TypeAck         WSPayloadType = "ack"        // every topic in the request succeeded
    TypeError       WSPayloadType = "error"
    TypePing        WSPayloadType = "ping"
    TypePong        WSPayloadType = "pong"
)
```

**Usage Examples** (v2.0+):

**Subscribe with ID** (client → server):
```json
{
    "type": "subscribe",
    "id": "req-123",
    "topic": ["users.*"]
}
```

**ACK Response** (server → client, v2.0+):
```json
{
    "type": "ack",
    "id": "req-123",
    "message": ""
}
```

**Publish with ID** (client → server):
```json
{
    "type": "publish",
    "id": "req-456",
    "topic": ["users.created"],
    "message": {
        "id": 123,
        "name": "John"
    }
}
```

**Event (fanout response)** (server → client):
```json
{
    "type": "event",
    "topic": ["users.created"],
    "message": {
        "id": 123,
        "name": "John"
    }
}
```

**Ping/Pong Heartbeat** (v2.0+):
```json
{
    "type": "ping"
}
```
Client responds:
```json
{
    "type": "pong"
}
```

**Error Response**:
```json
{
    "type": "error",
    "id": "req-123",
    "message": {
        "code": 404,
        "error": "not found"
    }
}
```

### 3.2 WSPayload Parsing

**Done automatically by framework**:
```
JSON string → WSPayload struct (via json.Unmarshal)
Parsed WSPayload available in handler via ctx.WSPayload
```

---

## 4. Topic Matching & Pattern Matching

### 4.1 Pattern Matching Algorithm

```
Topic: "users.created"

1. Exact match: "users.created" → handler found
   NO → continue
   
2. Suffix wildcard: "users.*" → handler found
   NO → continue
   
3. Complex regex: "users\.[a-z]+" → handler found
   NO → continue
   
4. Global fallback: "*" → handler found
   NO → 404 Not Found
```

### 4.2 Topic Pattern Examples

| Pattern | Matches | Doesn't Match |
|---------|---------|---------------|
| `users.created` | `users.created` | `users.updated`, `products.created` |
| `users.*` | `users.created`, `users.updated` | `users`, `products.created` |
| `*.created` | `users.created`, `products.created` | `users.updated`, `created` |
| `users.*.details` | `users.123.details` | `users.details`, `users.123.456.details` |
| `*` | Any topic | (nothing) |

---

## 5. Event Handler Registration

### 5.1 Handler Method Naming

**WS Controllers embed `common.WS`**:
```go
type ChatController struct {
    common.WS
}
```

**Handler methods = event names**:
```go
func (c ChatController) ON_MESSAGE_CREATED() string {
    // Routes to topic: "message.created"
    return "event received"
}

func (c ChatController) ON_ROOM_MESSAGE() string {
    // Routes to topic: "room.message"
    return "ok"
}
```

### 5.2 Method Name → Topic Conversion

**Algorithm**: Convert method name to topic

```
Method: ON_MESSAGE_CREATED
Step 1: Remove ON_ prefix → MESSAGE_CREATED
Step 2: Convert to lowercase → message_created
Step 3: Replace _ with . → message.created
Result Topic: "message.created"
```

---

## 6. Subscription & Broadcasting

### 6.1 Memorybroker.Subscribe()

**Purpose**: Register callback for topic pattern

**Pseudo-Code**:
```
memorybroker.Subscribe("users.*", func(data any) {
    // Called for any message matching "users.*"
    // Non-blocking, fire-and-forget
})
```

**Used by**: Fanout handlers (to broadcast to all subscribed connections)

### 6.2 Fanout Handler Pattern

**Generated automatically by framework**:
```go
memorybroker.Subscribe("users.created", func(data any) {
    // conn.TrySend(WSPayload{...})
    // Non-blocking: if buffer full, message drops
})
```

**Semantics**:
- One fanout handler per subscription per connection
- Called when memorybroker publishes to topic
- Responsible for sending to specific connection

### 6.3 Send Channel Semantics

**Per-Connection Buffer**:
- Size: `WSConfig.SendBufferSize` (default 32)
- Behavior: Non-blocking `TrySend()`
- Overflow: the frame is dropped, the drop is counted, and the first drop logs
  `WSSlowConsumer`
- Past `WSConfig.MaxDroppedFrames` (default 64) the connection is evicted

**Implication**: a slow client loses messages up to the drop budget and is then
disconnected, rather than degrading silently and indefinitely. `App.WSStats()`
exposes `Connections`, `Subscriptions`, `Dropped` and `SlowConsumers`.

**Write-side bounds** (these are what stop a slow client becoming a leak):
- `WSConfig.WriteTimeout` (default 10s) sets a deadline before every `websocket.JSON.Send`. Without it a peer that never reads blocks `Send` forever, and `close(done)` cannot free the goroutine because it is blocked inside `Send`, not in its `select`.
- A write error closes the underlying conn. That unblocks `readLoop`, which lets `handleRequest` run its deferred `Unregister`. Without it a half-broken connection stays registered and silently drops every outbound message — and the dead-conn reaper never reaps it, because `touch()` only fires on **read**, so a peer that keeps sending keeps `LastSeen` fresh.

See [security-limits-and-defaults.md](security-limits-and-defaults.md) for every cap and its opt-out.

---

## 7. WebSocket Dependencies (Injection)

### 7.1 Available Dependencies

```go
func (c ChatController) SUBSCRIBE_chat_ANY(
    wsCtx *ctx.WSContext,           // full context: ConnID(), Topic(), Pattern(), Operation()
    conn *websocket.Conn,           // raw connection
    payload ctx.WSPayload,          // current message
    topic ctx.WSTopic,              // the concrete topic being served, e.g. "chat.42"
    next ctx.Next,                  // middleware continuation
    pub common.Publisher,           // message publisher (fan-out from a handler)
) string {
    // All available
    return "handled"
}
```

**NOT Available in WS**:
- Body, Query, Header, Param, Form, File (HTTP-specific)
- http.Request, http.ResponseWriter (HTTP-specific)

### 7.2 WSPayload Pipeable

```go
type MessagePipeable struct {
    // Custom transformation of WSPayload
}

func (m MessagePipeable) Transform(raw any, metadata ArgumentMetadata) any {
    payload := raw.(ctx.WSPayload)
    // Validate, transform
    return transformedData
}

// Used in handler:
func (c ChatController) ON_MESSAGE_CREATED(data TransformedData) {
    // data is already transformed by Pipeable
}
```

---

## 8. Concurrency Model

### 8.1 Per-Connection Concurrency

**3 goroutines per connection** (2 spawned + the handler's own):
```
readLoop()  ← runs INLINE on the goroutine websocket.Handler gave handleRequest (NOT spawned)
writeLoop() ← spawned by WSConnmgr.Register; drains the 32-slot send channel
pingLoop()  ← spawned by handleRequest; 30s ticker

Handler execution ← runs in readLoop's goroutine

memorybroker.Publish() ← calls callbacks (non-blocking TrySend)
```

Total WS goroutines are bounded at ~3 x `WSConfig.MaxConnections` (default 10000).

**No Race Conditions Within Connection**:
- Only one message processed at a time
- Handler can safely access connection state
- Context is not shared with other connections

### 8.2 Global Memorybroker Concurrency

**Thread-Safe**:
- Memorybroker.Subscribe() — concurrent-safe
- Memorybroker.Publish() — concurrent-safe
- Internal: single sync.RWMutex over 4 maps (exact/prefix/global/complex); no sharding. Handlers are snapshotted into a private slice under the lock, then invoked after unlocking — a fanout callback may safely re-enter Subscribe/Unsubscribe/Publish/PublishAsync without deadlocking
- Subscriber panics are recovered per-handler by `callHandler` and reported (default: stderr; override with `NewMemoryBroker(WithPanicHandler(fn))`). One bad fanout callback cannot kill the publish loop

**Fanout Callbacks**:
- Called sequentially (one at a time)
- Each callback is fast (non-blocking TrySend)
- No guaranteed order across connections

---

## 9. Error Handling in WebSocket

### 9.1 Handshake Errors

```
Guard returns false in handshake
    ↓
Connection NOT accepted
    ↓
HTTP error response (403 Forbidden)
    ↓
WebSocket upgrade fails (no connection)
```

### 9.2 Handler Errors

```
Handler panics
    ↓
Panic recovered (same as HTTP)
    ↓
Exception filter called
    ↓
Filter sends error response via conn.Send()
```

### 9.3 Connection Errors

```
conn.Receive() returns error (connection closed, etc.)
    ↓
readLoop exits
    ↓
Unregister() triggered
    ↓
All resources cleaned up
```

---

## 10. WebSocket Close Handling

### 10.1 Close Codes

**Supported Close Codes**:
- 1000: Normal Closure
- 1001: Going Away
- 1002: Protocol Error
- 1003: Unsupported Data
- 1006: Abnormal Closure
- 1008: Policy Violation
- 1011: Server Error

### 10.2 Graceful Shutdown

```
app.Stop()
    ↓
Iterate all active connections
    ↓
Send close frame (code 1000)
    ↓
readLoop exits
    ↓
writeLoop exits
    ↓
Resources freed
```

---

## 11. Heartbeat & Liveness Detection

### 11.1 Ping-Pong Heartbeat

**Purpose**: Detect stale connections and keep TCP connection alive

**Mechanism**:
```
Server (pingLoop) sends TypePing every WSConfig.PingInterval (default 30s)
    ↓
Client receives TypePing
    ↓
Client sends TypePong response
    ↓
Server (readLoop) receives TypePong
    ↓
Server updates conn.lastSeen timestamp (any inbound frame does this)
```

A client may also drive the heartbeat itself: the server answers a client
`ping` with a `pong` **echoing the ping's `ID`**, so a client can match a pong
to the heartbeat it sent and time out on its own schedule.

**Guarantees**:
- If a client stops sending anything at all, the server reaps it within
  `WSConfig.PongTimeout` (default 75s), swept every `WSConfig.ReapInterval`
  (default 15s)
- No application-level heartbeat needed (framework handles it)
- Helps NAT/firewall keep TCP connection alive

**Caveat**: this is an *application-level* JSON `ping`, not a WebSocket
control frame (opcode 0x9). Browsers auto-answer protocol pings but will NOT
answer this one — a non-ginject client must reply with `{"type":"pong"}` or it
will be reaped. `ginject-sdk` does this for you.

### 11.2 Dead Connection Detection

**Purpose**: Clean up zombie connections from unresponsive clients

**Mechanism**:
```
startDeadConnDetection goroutine (runs every 15 seconds):
    1. Scan all connections
    2. Find any with LastSeen > 60 seconds old
    3. For each dead connection:
        ├─ Close underlying TCP connection
        ├─ Unsubscribe from all broker topics
        ├─ Close done channel
        └─ Remove from connection map
```

**Implementation Detail**:
```
RWMutex phase 1 (read-locked):
    └─ Collect IDs of dead connections

Per-connection phase (lock/unlock cycle):
    └─ Close connection
    └─ Unsubscribe
    └─ Clean up maps
    └─ Unlock, then next connection

Reason: Avoid holding lock during cleanup, prevent lock contention
```

---

## 12. Known Limitations & Gotchas

### 12.0 Single-Process Fan-Out

`memorybroker` is in-process. Two ginject instances behind a load balancer do
NOT see each other's publishes. `App.UseBroker(b memorybroker.Broker)` (or
`WSConfig.Broker`) swaps in a distributed implementation — the WebSocket layer
itself needs no changes — but a Redis/NATS/Kafka broker still has to be
written, and each has its own delivery semantics (at-most-once vs at-least-
once, ordering, consumer groups). Until one is plugged in, treat WebSocket
fan-out as single-instance.

### 12.1 Message Drop on Slow Client

```
GOTCHA: Send channel buffer is 32 messages
If client reads slowly, buffer fills
memorybroker.Publish() → TrySend() → buffer full → message DROPPED
Client never sees message, no error notification
```

**Mitigation**: Monitor client lag, implement backpressure

### 12.2 No Message Ordering Guarantee Across Connections

```
GOTCHA: Fanout sends to all subscribers
Order of delivery to different clients is undefined
Client A might see message before Client B
(implementation is actually ordered, but not guaranteed)
```

### 12.3 Fire-and-Forget Semantics

```
GOTCHA: Fanout callbacks are non-blocking
If fanout callback panics (unlikely), it's silently dropped
No error handling for individual subscriber failures
```

### 12.4 GOTCHA: Connection Hijacking

```
GOTCHA: Raw *websocket.Conn available in handler
If you read/write directly to conn, you bypass framework
Can cause message corruption or race conditions
```

**Rule**: Use Publisher, never access raw conn directly

---

## 13. Performance Considerations

### 13.1 Goroutine Count

**Per Active Connection**: 2 goroutines (readLoop + writeLoop)

**Example**: 10,000 active connections = 20,000 goroutines

**Cost**: ~50KB per goroutine (stack) = 1GB for 20K goroutines

### 13.2 Memory Usage

**Per Connection**:
- WSContext: ~2KB
- Send channel: ~32 × WSPayload = ~10KB
- Connection state: ~1KB
- Total: ~15KB per connection

**Example**: 10,000 connections = 150MB

### 13.3 Message Throughput

**Publish throughput**: not benchmarked end-to-end (JSON marshal + TCP send dominate at the WS layer, unmeasured here). `memorybroker.Publish` itself is benchmarked in isolation (`memorybroker/broker_bench_test.go`): ~18-24 µs/op fanning out to 1000 exact-topic subscribers, ~2 µs/op to 10 mixed exact+prefix+global subscribers, 0 allocs when a topic has no matching subscriber — see that package's README for current numbers before citing a figure

**Per-connection throughput**: Limited by JSON marshaling/unmarshaling

**Bottleneck**: writeLoop serialization + TCP send

---

## 14. WebSocket Best Practices

### 14.1 DO

- ✓ Use Publisher to send messages (async, non-blocking)
- ✓ Keep handlers short and fast
- ✓ Use topic patterns for selective delivery
- ✓ Monitor connection count and memory
- ✓ Implement heartbeat/ping-pong for stale connections (framework does this automatically now)

### 14.2 DON'T

- ✗ Access raw conn.Send()/Receive() in handlers
- ✗ Hold onto connection reference after handler
- ✗ Broadcast to all connections (use topics instead)
- ✗ Assume FIFO ordering across connections
- ✗ Assume all messages are delivered

