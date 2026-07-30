# WebSocket System & Pub/Sub Broker

**Optimization**: Connection lifecycle state machine, pub/sub decision trees, concurrency guarantees.

## 1. WebSocket Connection Lifecycle

### 1.1 State Machine

```
[INIT]
  ↓
[HANDSHAKE] — Middleware chain runs
  ├─ Guard checks if handshake allowed
  └─ Can set initial context
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
        // Middleware chain runs here
        // Guard can reject: return error
        // Interceptor can setup context
        return nil  // Accept connection
    },
    Handler: websocket.Handler(app.ws.handleRequest),
})
```

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
    └─ Handler can broker.Publish()

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
All broker subscriptions cleaned up
    ↓
Connection resources freed
```

---

## 2. Message Flow

### 2.1 Inbound Message Processing

```
readLoop() receives raw message
    ↓
Unmarshal JSON → WSPayload{Type, Topic, Data, ...}
    ↓
Pattern matching: Topic
    ├─ Exact match → get handler
    ├─ Suffix wildcard (*) → get handler
    ├─ Complex regex → get handler
    ├─ Global fallback → get handler
    └─ No match → 404 (error response)
    ↓
Handler resolution (dependency injection)
    ↓
Pipeline execution:
    ├─ Middleware (same as HTTP)
    ├─ Guard (same as HTTP, but WS-specific)
    ├─ Interceptor (pre)
    ├─ Handler execution
    ├─ Interceptor (post)
    └─ Exception filter (if panic)
    ↓
Response (if handler returns data)
    ↓
handler can call broker.Publish(topic, data)
    ↓
Fanout to all subscribers
```

### 2.2 Outbound Message Broadcasting

```
broker.Publish("users.created", userData)
    ↓
Broker looks up subscribers for "users.created"
    ↓
For each subscriber connection:
    ├─ Get subscription callback (fanout handler)
    ├─ Call callback(userData)
    └─ Callback calls conn.TrySend(message) ← NON-BLOCKING
    ↓
If send channel buffer full (32 messages):
    ├─ TrySend() returns false
    ├─ Message DROPPED silently
    └─ No error/exception thrown
```

---

## 3. WebSocket Payload Structure

### 3.1 WSPayload

```go
type WSPayload struct {
    Type    string                 // "publish", "subscribe", "unsubscribe"
    Topic   string                 // Topic name or pattern
    Data    any                    // Message data (marshals to/from JSON)
    Message map[string]interface{} // Auxiliary message fields
}
```

**Usage Examples**:

**Publish (handler sends to others)**:
```json
{
    "type": "publish",
    "topic": "users.created",
    "data": {
        "id": 123,
        "name": "John"
    }
}
```

**Subscribe (client subscribes to topic)**:
```json
{
    "type": "subscribe",
    "topic": "users.*"
}
```

**Unsubscribe**:
```json
{
    "type": "unsubscribe",
    "topic": "users.*"
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

### 6.1 Broker.Subscribe()

**Purpose**: Register callback for topic pattern

**Pseudo-Code**:
```
broker.Subscribe("users.*", func(data any) {
    // Called for any message matching "users.*"
    // Non-blocking, fire-and-forget
})
```

**Used by**: Fanout handlers (to broadcast to all subscribed connections)

### 6.2 Fanout Handler Pattern

**Generated automatically by framework**:
```go
broker.Subscribe("users.created", func(data any) {
    // conn.TrySend(WSPayload{...})
    // Non-blocking: if buffer full, message drops
})
```

**Semantics**:
- One fanout handler per subscription per connection
- Called when broker publishes to topic
- Responsible for sending to specific connection

### 6.3 Send Channel Semantics

**Per-Connection Buffer**:
- Size: 32 messages (fixed)
- Behavior: Non-blocking TrySend()
- Overflow: Messages DROPPED silently
- No backpressure mechanism

**Implication**: Slow clients can miss messages

---

## 7. WebSocket Dependencies (Injection)

### 7.1 Available Dependencies

```go
func (c ChatController) ON_MESSAGE_CREATED(
    wsCtx *ctx.WSContext,           // Full context
    conn *websocket.Conn,           // Raw connection
    payload ctx.WSPayload,          // Current message
    next ctx.Next,                  // Middleware continuation
    pub common.Publisher,           // Message publisher
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

**Single-Threaded Per Connection**:
```
readLoop() ← 1 goroutine (blocking on receive)
writeLoop() ← 1 goroutine (blocking on send channel)

Handler execution ← Runs in readLoop's goroutine

broker.Publish() ← Calls callbacks (non-blocking TrySend)
```

**No Race Conditions Within Connection**:
- Only one message processed at a time
- Handler can safely access connection state
- Context is not shared with other connections

### 8.2 Global Broker Concurrency

**Thread-Safe**:
- Broker.Subscribe() — concurrent-safe
- Broker.Publish() — concurrent-safe
- Internal: sync.RWMutex for subscriber list

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

## 11. Known Limitations & Gotchas

### 11.1 Message Drop on Slow Client

```
GOTCHA: Send channel buffer is 32 messages
If client reads slowly, buffer fills
broker.Publish() → TrySend() → buffer full → message DROPPED
Client never sees message, no error notification
```

**Mitigation**: Monitor client lag, implement backpressure

### 11.2 No Message Ordering Guarantee Across Connections

```
GOTCHA: Fanout sends to all subscribers
Order of delivery to different clients is undefined
Client A might see message before Client B
(implementation is actually ordered, but not guaranteed)
```

### 11.3 Fire-and-Forget Semantics

```
GOTCHA: Fanout callbacks are non-blocking
If fanout callback panics (unlikely), it's silently dropped
No error handling for individual subscriber failures
```

### 11.4 Connection Hijacking

```
GOTCHA: Raw *websocket.Conn available in handler
If you read/write directly to conn, you bypass framework
Can cause message corruption or race conditions
```

**Rule**: Don't use raw conn.Send()/Receive() in handlers, use Publisher instead

---

## 12. Performance Considerations

### 12.1 Goroutine Count

**Per Active Connection**: 2 goroutines (readLoop + writeLoop)

**Example**: 10,000 active connections = 20,000 goroutines

**Cost**: ~50KB per goroutine (stack) = 1GB for 20K goroutines

### 12.2 Memory Usage

**Per Connection**:
- WSContext: ~2KB
- Send channel: ~32 × WSPayload = ~10KB
- Connection state: ~1KB
- Total: ~15KB per connection

**Example**: 10,000 connections = 150MB

### 12.3 Message Throughput

**Publish throughput**: 100K-1M messages/sec (broker + fanout)

**Per-connection throughput**: Limited by JSON marshaling/unmarshaling

**Bottleneck**: writeLoop serialization + TCP send

---

## 13. WebSocket Best Practices

### 13.1 DO

- ✓ Use Publisher to send messages (async, non-blocking)
- ✓ Keep handlers short and fast
- ✓ Use topic patterns for selective delivery
- ✓ Monitor connection count and memory
- ✓ Implement heartbeat/ping-pong for stale connections

### 13.2 DON'T

- ✗ Access raw conn.Send()/Receive() in handlers
- ✗ Hold onto connection reference after handler
- ✗ Broadcast to all connections (use topics instead)
- ✗ Assume FIFO ordering across connections
- ✗ Assume all messages are delivered

