# Ginject AI Knowledge Base

**Comprehensive, machine-optimized framework documentation for AI code generation agents.**

---

## Overview

This directory (`.ginject/`) contains complete AI-friendly documentation of the Ginject framework, optimized for machine understanding and code generation.

**Key Characteristics**:
- ✅ **Complete Coverage**: 20+ packages, 100+ public APIs, all core concepts
- ✅ **Machine-Readable**: Decision trees, state machines, JSON metadata
- ✅ **Anti-Pattern Documentation**: What NOT to do, with examples
- ✅ **Decision Trees**: Quick guidance for common scenarios
- ✅ **Lifecycle Documentation**: Exact state transitions, initialization order
- ✅ **Concurrency Model**: Thread safety guarantees, race condition scenarios
- ✅ **Performance Analysis**: Latency breakdowns, bottlenecks, optimization tips

---

## Files at a Glance

### Start Here

| File | Purpose | Read If |
|------|---------|---------|
| **INDEX.md** | Navigation hub | First time here, need overview |
| **ai-guidance-quick-reference.md** | Decision trees for common tasks | Need to implement a feature |

### Core Architecture

| File | Purpose | Read If |
|------|---------|---------|
| **architecture-core-concepts.md** | Foundation concepts | Want to understand how Ginject works |
| **request-pipeline.md** | Request lifecycle and pipeline | Implementing middleware, guards, or filters |
| **handler-execution.md** | Handler invocation and DI | Need to understand parameter injection |
| **routing-system.md** | URL routing and naming | Implementing HTTP endpoints |
| **websocket-system.md** | WebSocket connections and pub/sub | Implementing WebSocket features |
| **modules-and-providers.md** | Module system and composition | Organizing application code |

### Reference

| File | Purpose | Read If |
|------|---------|---------|
| **package-reference.md** | Complete API reference | Need exact method signatures |
| **anti-patterns-gotchas.md** | Common mistakes and what NOT to do | Debugging issues or reviewing code |

### Machine-Readable Metadata

| File | Purpose | Read If |
|------|---------|---------|
| **metadata-api-index.json** | Structured API index | AI agent needs structured data |
| **metadata-concurrency-model.json** | Concurrency guarantees | Analyzing thread safety |
| **metadata-performance.json** | Performance characteristics | Performance optimization |
| **metadata-lifecycles.json** | State machines and lifecycle | Understanding initialization |

---

## Key Takeaways (30-Second Version)

### Architecture
- **Module → Controller → Provider**: Composition pattern
- **Pipeline**: Middleware → Guard → Interceptor → Handler → Filter
- **Reflection-based DI**: Type-matched, no tags needed

### Critical Rules (NEVER violate)
1. Never hold context after request
2. Never panic in guard
3. Never call next() multiple times
4. Never call NewProvider() manually
5. Module init order is SACRED

### Concurrency
- HTTP: Each request is independent (thread-safe)
- WebSocket: Each connection is single-threaded (2 goroutines: read + write)
- Global state: Read-only after app.Create()

### Performance
- Typical request: 1-2ms (simple handler, no I/O)
- Dependency injection: 5-10μs per parameter
- Reflection overhead: main bottleneck

---

## Using This Knowledge Base

### For Developers
```
1. Read INDEX.md → navigation hub
2. Read architecture-core-concepts.md → understand framework
3. Jump to specific topics (routing, WebSocket, modules, etc.)
4. Reference package-reference.md for exact APIs
5. Check anti-patterns-gotchas.md before implementing
```

### For AI Code Generators
```
1. Load ai-guidance-quick-reference.md
2. Use decision trees to determine approach
3. Reference metadata-*.json for structured data
4. Verify against anti-patterns-gotchas.md
5. Generate code using package-reference.md
```

### For Code Reviewers
```
1. Check against anti-patterns-gotchas.md for violations
2. Verify pipeline order matches request-pipeline.md
3. Confirm concurrency safety with metadata-concurrency-model.json
4. Check initialization order against metadata-lifecycles.json
```

---

## Document Organization

### By Concept
```
Core Framework:
  → architecture-core-concepts.md (foundational)
  → request-pipeline.md (request execution)
  → handler-execution.md (handler invocation)

Features:
  → routing-system.md (HTTP routing)
  → websocket-system.md (WebSocket pub/sub)
  → modules-and-providers.md (composition)

Guidance:
  → ai-guidance-quick-reference.md (decision trees)
  → anti-patterns-gotchas.md (what NOT to do)

Reference:
  → package-reference.md (all public APIs)
  → metadata-*.json (structured data)
```

### By Topic
```
Dependency Injection:
  → handler-execution.md (parameter injection)
  → modules-and-providers.md (provider system)
  → package-reference.md (Provider interface)

Error Handling:
  → request-pipeline.md (exception filter layer)
  → anti-patterns-gotchas.md (panic rules)
  → architecture-core-concepts.md (recovery points)

WebSocket:
  → websocket-system.md (complete reference)
  → ai-guidance-quick-reference.md (quick start)
  → anti-patterns-gotchas.md (common mistakes)

Performance:
  → metadata-performance.json (latency, bottlenecks)
  → architecture-core-concepts.md (design decisions)
  → anti-patterns-gotchas.md (what slows things down)
```

---

## Common Scenarios → Quick Links

| Scenario | Read |
|----------|------|
| "Create HTTP endpoint" | routing-system.md + ai-guidance-quick-reference.md |
| "Add authentication" | request-pipeline.md (guard section) + ai-guidance-quick-reference.md |
| "Add validation" | ai-guidance-quick-reference.md (validation decision tree) |
| "Add WebSocket" | websocket-system.md + ai-guidance-quick-reference.md |
| "Create service/provider" | modules-and-providers.md + handler-execution.md |
| "Handle errors" | request-pipeline.md (filter section) + ai-guidance-quick-reference.md |
| "Debug issue" | anti-patterns-gotchas.md + metadata-lifecycles.json |
| "Optimize performance" | metadata-performance.json + architecture-core-concepts.md |

---

## Key Concepts Explained

### 1. Module System
**What**: Composition root for controllers, providers, and settings
**How**: Declare controllers, providers, and imports; register via ModuleBuilder
**When**: Set up once at app.Create(), immutable after
**Example**:
```go
module := ModuleBuilder().
    Controllers(UserController{}).
    Providers(UserService{}, UserRepository{}).
    Imports(AuthModule).
    Build()
app.Create(module)
```

### 2. Dependency Injection
**What**: Automatic parameter resolution by type matching
**How**: Declare parameter in handler signature; framework injects by type
**When**: Per-request for controllers, per-injection for providers
**Example**:
```go
func (c UserController) READ(httpCtx *ctx.HTTPContext, us UserService) string {
    // httpCtx and us injected automatically
    return us.GetUser(httpCtx.Param.Get("id"))
}
```

### 3. Request Pipeline
**What**: Sequential execution of middleware, guards, interceptors, handlers, filters
**How**: Bound to controllers or app
**When**: Every HTTP request or WebSocket message
**Example**:
```
Global Middleware → Module Middleware → Guard → Handler → Exception Filter
```

### 4. WebSocket Pub/Sub
**What**: Topic-based message fanout to all subscribers
**How**: Broker.Publish() to topic, all subscribed connections receive
**When**: Handler can publish, or external service publishes
**Example**:
```go
publisher.Publish("users.created", userData)  // All subscribers receive
```

### 5. Exception Handling
**What**: Panic recovery and error response writing
**How**: Panic in handler caught, exception filter writes response
**When**: Handler panics or validation fails
**Example**:
```go
panic(exception.BadRequestException("invalid"))  // Caught → 400 response
```

---

## Critical Rules (Remember These)

### Never
- ❌ Hold context after request ends
- ❌ Mutate context after guard
- ❌ Panic in guard (return bool instead)
- ❌ Call next() multiple times in middleware
- ❌ Call NewProvider() manually
- ❌ Access raw websocket.Conn in handlers
- ❌ Create circular dependencies
- ❌ Reorder app.Create() initialization steps

### Always
- ✅ Inject dependencies via handler parameters
- ✅ Use middleware for request/response transformation
- ✅ Use guard for authentication/authorization
- ✅ Use exception filter for error recovery
- ✅ Use Publisher for WebSocket fanout
- ✅ Make providers singletons (return same instance)
- ✅ Call next() exactly once per middleware

---

## Performance Tips

From [metadata-performance.json](metadata-performance.json):

**Typical fast path**: 1-2ms (simple handler)
- Context pooling: 500ns
- Route matching: 10-100μs
- Handler params: 25-50μs (5 deps × 5-10μs each)
- Middleware: 20-60μs
- Handler execution: 500-1000μs

**Optimization**:
- Minimize handler parameters (reflect cost)
- Avoid exceptions in hot path
- Keep middleware count low
- Use binary serialization for high throughput

---

## Threading Model

**HTTP Requests**:
- 1 goroutine per request (in request's goroutine)
- Each request fully isolated (no shared state)
- Thread-safe by design

**WebSocket Connections**:
- 2 goroutines per connection (readLoop + writeLoop)
- Connection single-threaded (no race within connection)
- Global broker thread-safe (sync.RWMutex)

**Global State**:
- Read-only after app.Create()
- Immutable (no mutations)
- All concurrent operations safe

---

## Troubleshooting Guide

| Problem | Likely Cause | Read |
|---------|-------------|------|
| "Dependency not found" | Type mismatch between provider and parameter | handler-execution.md#1.1 |
| "Stack overflow at startup" | Circular dependency | modules-and-providers.md#5.3 |
| "Context has wrong data" | Holding context after request | anti-patterns-gotchas.md#2.1 |
| "Handler called twice" | Calling next() multiple times | anti-patterns-gotchas.md#3.2 |
| "Slow client drops messages" | WS send buffer overflow | websocket-system.md#11.1 |
| "Request takes too long" | No timeout set or handler hanging | architecture-core-concepts.md#3.2 |
| "Messages corrupt" | Concurrent writes to WS connection | websocket-system.md#4.2 |
| "Memory usage high" | Retaining pooled contexts | anti-patterns-gotchas.md#2.1 |

---

## For Different Roles

### Backend Developer
```
Read: architecture-core-concepts.md → routing-system.md → handler-execution.md
Then: ai-guidance-quick-reference.md (for common tasks)
Reference: package-reference.md (for APIs)
```

### DevOps / Infrastructure
```
Read: metadata-performance.json (throughput, capacity)
Then: metadata-concurrency-model.json (goroutine limits)
Reference: architecture-core-concepts.md#6-concurrency-model
```

### QA / Tester
```
Read: anti-patterns-gotchas.md (what breaks)
Then: metadata-lifecycles.json (state transitions)
Reference: ai-guidance-quick-reference.md (common scenarios)
```

### Security Reviewer
```
Read: anti-patterns-gotchas.md (vulnerabilities)
Then: metadata-concurrency-model.json (race conditions)
Reference: request-pipeline.md#3-guard-layer (authorization)
```

---

## Document Statistics

| Category | Count | Coverage |
|----------|-------|----------|
| Markdown Documents | 7 | Complete architecture documentation |
| JSON Metadata Files | 4 | Structured data for AI parsing |
| Core Packages Documented | 20+ | All major packages |
| Public APIs Documented | 100+ | All public types and methods |
| Code Patterns Documented | 50+ | Architecture patterns and examples |
| Anti-Patterns Documented | 20+ | What NOT to do |
| Decision Trees | 10+ | Common scenario guidance |
| Total Pages (Markdown) | 50+ | Comprehensive reference |

---

## Maintenance & Updates

**Last Updated**: July 2026  
**Coverage**: Ginject pre-v1.0 (6.2/10 production readiness)  
**Source**: Complete source code analysis + ARCHITECTURE_REVIEW_AND_ROADMAP.md

**To Update This Knowledge Base**:
1. Verify against current source code in /Users/tadangduoc/Projects/ginject
2. Update affected markdown files
3. Update corresponding metadata-*.json
4. Update this README with new files
5. Commit with message: "docs: update AI knowledge base"

---

## Next Steps for AI Agents

**If you're Claude Code or another AI agent**:

1. **Load Context**: Read INDEX.md and ai-guidance-quick-reference.md
2. **Choose Approach**: Use decision trees to pick implementation strategy
3. **Verify Safety**: Check anti-patterns-gotchas.md for violations
4. **Generate Code**: Reference package-reference.md for exact APIs
5. **Review**: Test against critical rules before submitting

**If you're a developer**:

1. **Understand**: Read architecture-core-concepts.md
2. **Go Deep**: Read feature-specific docs (routing, WebSocket, modules)
3. **Implement**: Use ai-guidance-quick-reference.md for quick guidance
4. **Reference**: Use package-reference.md for exact APIs
5. **Verify**: Check anti-patterns-gotchas.md before committing

---

## License & Attribution

This knowledge base is derived from the Ginject framework source code.

**Framework**: Ginject (NestJS-inspired dependency injection web framework for Go)  
**Author**: dangduoc08  
**Repository**: https://github.com/dangduoc08/ginject  
**Documentation**: AI-Optimized (July 2026)

---

## Questions?

**For developers**: Refer to the main CLAUDE.md in repository root  
**For AI agents**: Prioritize decision trees in ai-guidance-quick-reference.md  
**For framework issues**: Check anti-patterns-gotchas.md first  
**For performance**: Consult metadata-performance.json

---

**This is your guide to building with Ginject. Trust it, use it, refer to it.**

