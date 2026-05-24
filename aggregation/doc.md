# Package `aggregation`

The `aggregation` package implements the post-handler data pipeline for ginject interceptors. It provides:

- A container (`Aggregation`) that holds the result produced by an HTTP/WebSocket handler.
- A set of **registered operators** (Consume, Map, Of, Error, First) that the framework reads back by name after the interceptor returns.
- A **pipeline** of composable, RxJS-style operator functions that transform, filter, recover from errors, or combine values before the response is written.

---

## Table of Contents

1. [Framework lifecycle](#framework-lifecycle)
2. [Types](#types)
3. [Constants](#constants)
4. [Constructor](#constructor)
5. [Aggregation methods](#aggregation-methods)
   - [Pipe](#pipe)
   - [Consume](#consume)
   - [Map (method)](#map-method)
   - [Of (method)](#of-method)
   - [Error](#error)
   - [First](#first)
   - [SetMainData](#setmaindata)
   - [GetAggregationOperator](#getaggregationoperator)
   - [Aggregate](#aggregate)
6. [Pipeline operator functions](#pipeline-operator-functions)
   - [Error propagation model](#error-propagation-model)
   - [Map](#map)
   - [Tap](#tap)
   - [CatchError](#catcherror)
   - [SwitchMap](#switchmap)
   - [MergeMap](#mergemap)
   - [ConcatMap](#concatmap)
   - [Of (function)](#of-function)
   - [From](#from)
   - [Finalize](#finalize)
   - [Filter](#filter)
   - [Take](#take)
   - [ThrowError](#throwerror)
   - [Delay](#delay)
   - [Timeout](#timeout)
   - [CombineLatest](#combinelatest)
   - [ForkJoin](#forkjoin)
   - [Scan](#scan)
7. [Use cases](#use-cases)

---

## Framework lifecycle

The framework creates a fresh `*Aggregation` for every intercepted request:

```
1. aggregationInstance := NewAggregation()

2. value := interceptor.Intercept(c, aggregationInstance)
      └─ interceptor calls aggregationInstance.Pipe(...)  ← operators stored, IsMainHandlerCalled = true
         or aggregationInstance.Consume(fn)               ← operator stored by name

3. aggregationInstance.InterceptorData = value

4. [if IsMainHandlerCalled] main handler executes, produces result

5. aggregationInstance.SetMainData(result)

6. finalValue := aggregationInstance.Aggregate(c)
      └─ runs Consume operator (if registered)
         then runs Pipe operators in order
         then unwraps any uncaught pipeline error to the underlying error value

7. finalValue is written as the HTTP response body
```

Multiple interceptors stack in LIFO order: the innermost interceptor's `Aggregate` runs first, and its result becomes the `mainData` of the next outer interceptor.

---

## Types

### `AggregationOperator`

```go
type AggregationOperator = func(*ctx.Context, any) any
```

The fundamental unit of the pipeline. Every operator — whether registered by name or composed in `Pipe` — has this signature.

- `*ctx.Context` — the current request context (HTTP or WebSocket).
- First `any` — the value flowing through the pipeline at this step.
- Returns `any` — the transformed value (or a pipeline error sentinel; see [Error propagation model](#error-propagation-model)).

### `Aggregation`

```go
type Aggregation struct {
    IsMainHandlerCalled bool
    InterceptorData     any
    // unexported: mainData, operators, pipeOperators
}
```

| Field | Access | Description |
|-------|--------|-------------|
| `IsMainHandlerCalled` | read/write | Set to `true` by `Pipe`. The framework checks this to decide whether to invoke the main handler after `Intercept` returns. |
| `InterceptorData` | read | The raw value returned by `Intercept`. The framework assigns this immediately after `Intercept` returns. |

---

## Constants

Defined in `operation.go`. Used as keys for the registered-operator map.

```go
const (
    OPERATOR_MAP                    = "Map"
    OPERATOR_OF                     = "Of"
    OPERATOR_CONSUME                = "Consume"
    OPERATOR_FIRST                  = "First"
    OPERATOR_ERROR                  = "Error"
    ERROR_AGGREGATION_CTX_VALUE_KEY = "ErrorAggregationOperators"
)
```

| Constant | Used by |
|----------|---------|
| `OPERATOR_CONSUME` | `Aggregate` — runs this operator on `mainData` before pipe operators |
| `OPERATOR_MAP` | Framework via `GetAggregationOperator` |
| `OPERATOR_OF` | Framework via `GetAggregationOperator` |
| `OPERATOR_FIRST` | Framework via `GetAggregationOperator` (stored as nil, acts as a flag) |
| `OPERATOR_ERROR` | Framework — collected into the request context; called from `recover()` on panic |
| `ERROR_AGGREGATION_CTX_VALUE_KEY` | Framework — context key under which error operators are stored |

---

## Constructor

### `NewAggregation`

```go
func NewAggregation() *Aggregation
```

Allocates a new `Aggregation` with an empty operator map (pre-sized to 5 entries). Called by the framework once per intercepted request; interceptors never call this directly.

---

## Aggregation methods

These are methods on `*Aggregation` that an interceptor calls inside `Intercept`.

---

### `Pipe`

```go
func (aggregation *Aggregation) Pipe(operators ...AggregationOperator) any
```

Signals that the main handler **should** run, and registers the provided pipeline operators to be applied in order when `Aggregate` is later called.

**Behaviour:**
- Sets `IsMainHandlerCalled = true`.
- Appends `operators` to the internal `pipeOperators` slice.
- Returns `nil` (the return value is intentionally discarded; `Intercept` should `return` the result of `Pipe` so the framework captures `InterceptorData = nil`).

**When to use:** Always call `Pipe` when the interceptor wants the request to reach the main handler. Operators passed to `Pipe` execute after the handler returns, transforming its result.

```go
func (i MyInterceptor) Intercept(c ginject.Context, agg ginject.Aggregation) any {
    return agg.Pipe(
        Map(func(c ginject.Context, data any) any {
            return map[string]any{"data": data}
        }),
    )
}
```

---

### `Consume`

```go
func (aggregation *Aggregation) Consume(opr AggregationOperator) AggregationOperator
```

Registers `opr` under the key `OPERATOR_CONSUME`. If a Consume operator is already registered, the second call is silently ignored (first-write-wins).

During `Aggregate`, the Consume operator runs **first**, before any Pipe operators. It is the primary hook for transforming the raw handler result.

Returns the operator as-is (for fluent use inside `Pipe`).

```go
return agg.Pipe(
    agg.Consume(func(c ginject.Context, data any) any {
        return map[string]any{"data": data, "ok": true}
    }),
)
```

**Note:** `Consume` is a method that registers an operator by name. It is different from the package-level `Map(fn)` / `Filter(fn)` functions, which are anonymous pipeline operators not addressable by name.

---

### `Map` (method)

```go
func (aggregation *Aggregation) Map(opr AggregationOperator) AggregationOperator
```

Registers `opr` under the key `OPERATOR_MAP`. The framework may retrieve it later via `GetAggregationOperator(OPERATOR_MAP)`. First-write-wins.

Returns the operator.

> **Distinct from the package-level `Map` function.** The method stores the operator in the named map; the function (`Map(fn)`) returns an anonymous pipeline operator for use in `Pipe`.

---

### `Of` (method)

```go
func (aggregation *Aggregation) Of(opr AggregationOperator) AggregationOperator
```

Registers `opr` under the key `OPERATOR_OF`. First-write-wins. Returns the operator.

> **Distinct from the package-level `Of` function.** The method registers a named operator; the function (`Of(val)`) is a pipeline operator that replaces the flowing value with a static `val`.

---

### `Error`

```go
func (aggregation *Aggregation) Error(opr AggregationOperator) AggregationOperator
```

Registers `opr` under the key `OPERATOR_ERROR`. First-write-wins.

The Error operator is **not** called during `Aggregate`. Instead, the framework collects all registered Error operators from all interceptors into the request context under `ERROR_AGGREGATION_CTX_VALUE_KEY`. When a panic occurs anywhere in the request pipeline, the framework's `recover()` block calls each collected Error operator with the panicking value, allowing interceptors to shape the error response.

```go
agg.Error(func(c ginject.Context, data any) any {
    return map[string]any{"error": fmt.Sprint(data)}
})
```

---

### `First`

```go
func (aggregation *Aggregation) First() AggregationOperator
```

Registers `nil` under the key `OPERATOR_FIRST`. Acts as a boolean flag that the framework can check with `GetAggregationOperator(OPERATOR_FIRST)`. Returns `nil`.

---

### `SetMainData`

```go
func (aggregation *Aggregation) SetMainData(d any) *Aggregation
```

Sets the value to be processed by `Aggregate`. Called exclusively by the framework after the main handler returns. Returns `*Aggregation` for chaining.

Interceptors should not call this directly.

---

### `GetAggregationOperator`

```go
func (aggregation *Aggregation) GetAggregationOperator(oprName string) AggregationOperator
```

Looks up a named operator (registered via `Consume`, `Map`, `Of`, `Error`, or `First`) by its constant key. Returns `nil` if the key is not present.

Used by the framework to retrieve the Error operator. Can also be used in tests or custom middleware to inspect what operators an interceptor registered.

```go
errorOpr := agg.GetAggregationOperator(aggregation.OPERATOR_ERROR)
if errorOpr != nil {
    // handle error
}
```

---

### `Aggregate`

```go
func (aggregation *Aggregation) Aggregate(c *ctx.Context) any
```

Executes the full post-handler pipeline and returns the final value. Called by the framework — interceptors never call this directly.

**Execution order:**
1. If `OPERATOR_CONSUME` is registered, call it with `mainData`.
2. Execute each Pipe operator in the order they were passed to `Pipe`.
3. If the final value is an internal pipeline error (`*pipeErr`), unwrap and return the underlying `error`. Otherwise return the value as-is.

---

## Pipeline operator functions

Pipeline operators are **package-level functions** that return an `AggregationOperator`. They are designed to be composed inside `Pipe`.

### Error propagation model

Errors in the pipeline are represented by the internal type `*pipeErr`. This is an opaque sentinel; user code never constructs it directly — only `ThrowError` and `Filter` (when the predicate fails) produce one.

**Rules for all operators:**

| Operator | On normal value | On `*pipeErr` |
|----------|-----------------|----------------|
| `Map`, `Tap`, `SwitchMap`, `MergeMap`, `ConcatMap` | apply transformation | pass through unchanged |
| `Filter`, `Take` | apply predicate/check | pass through unchanged |
| `CatchError` | pass through unchanged | call recovery fn, return result |
| `Finalize` | run side-effect, pass through | run side-effect, pass through |
| `ThrowError`, `Of`, `From` | replace with new value/error | replace (unconditionally) |
| `CombineLatest`, `ForkJoin`, `Scan` | apply combination/accumulation | pass through unchanged |

At the end of `Aggregate`, any uncaught `*pipeErr` is unwrapped to its underlying `error` and returned to the framework as the handler's result.

---

### `Map`

```go
func Map(fn func(*ctx.Context, any) any) AggregationOperator
```

Applies `fn` to the current value and replaces it with `fn`'s return. Skips `fn` when a pipeline error is flowing.

```go
agg.Pipe(
    Map(func(c ginject.Context, data any) any {
        user := data.(User)
        return UserDTO{ID: user.ID, Name: user.Name}
    }),
)
```

---

### `Tap`

```go
func Tap(fn func(*ctx.Context, any)) AggregationOperator
```

Runs `fn` as a side-effect (logging, metrics, tracing) and passes the original value through unchanged. Skips when a pipeline error is flowing.

```go
agg.Pipe(
    Tap(func(c ginject.Context, data any) {
        log.Printf("handler returned: %v", data)
    }),
)
```

---

### `CatchError`

```go
func CatchError(fn func(*ctx.Context, error) any) AggregationOperator
```

Intercepts a pipeline error and calls `fn` with the underlying `error`. The value returned by `fn` replaces the error and resumes normal flow. If no error is present, the value passes through unchanged.

`fn` may return a new error value by calling `ThrowError` inside it, or may recover by returning a normal value.

```go
agg.Pipe(
    ThrowError(someErr),
    CatchError(func(c ginject.Context, err error) any {
        return map[string]any{"error": err.Error()}
    }),
)
```

---

### `SwitchMap`

```go
func SwitchMap(fn func(*ctx.Context, any) any) AggregationOperator
```

Projects the current value to a new value via `fn`. Semantically equivalent to `Map` in a synchronous single-value pipeline. Skips on pipeline error.

Use `SwitchMap` over `Map` to signal intent: the projection may conceptually cancel a previous projection (mirroring RxJS).

```go
agg.Pipe(
    SwitchMap(func(c ginject.Context, data any) any {
        id := data.(string)
        return fetchLatestState(id) // only cares about the latest
    }),
)
```

---

### `MergeMap`

```go
func MergeMap(fn func(*ctx.Context, any) any) AggregationOperator
```

Projects the current value to a new value via `fn`. Semantically equivalent to `Map` in a synchronous single-value pipeline. Skips on pipeline error.

Use `MergeMap` to signal that the projection result may represent a merged/flattened collection.

```go
agg.Pipe(
    MergeMap(func(c ginject.Context, data any) any {
        items := data.([]Item)
        return flattenItems(items)
    }),
)
```

---

### `ConcatMap`

```go
func ConcatMap(fn func(*ctx.Context, any) any) AggregationOperator
```

Projects the current value to a new value via `fn`. Semantically equivalent to `Map` in a synchronous single-value pipeline. Skips on pipeline error.

Use `ConcatMap` to signal that projections should be applied sequentially (mirroring RxJS).

---

### `Of` (function)

```go
func Of(val any) AggregationOperator
```

Returns an operator that **ignores** the incoming value (including pipeline errors) and unconditionally emits `val`. Useful for replacing the pipeline value with a static constant.

```go
agg.Pipe(
    Of(map[string]any{"status": "ok"}),
)
```

---

### `From`

```go
func From(values []any) AggregationOperator
```

Returns an operator that **ignores** the incoming value and emits `values` as `[]any`. Useful for replacing the pipeline value with a fixed collection.

```go
agg.Pipe(
    From([]any{"admin", "editor", "viewer"}),
)
```

---

### `Finalize`

```go
func Finalize(fn func(*ctx.Context)) AggregationOperator
```

Runs `fn` as a side-effect **regardless** of whether the pipeline is in a normal or error state, then passes the current value (or error) through unchanged. Mirrors RxJS `finalize` / `finally`.

`fn` does not receive the pipeline value; it is a no-arg teardown callback.

```go
agg.Pipe(
    ThrowError(ErrNotFound),
    Finalize(func(c ginject.Context) {
        metrics.RecordRequestDone() // always runs
    }),
    CatchError(func(c ginject.Context, err error) any {
        return map[string]any{"error": err.Error()}
    }),
)
```

---

### `Filter`

```go
func Filter(predicate func(*ctx.Context, any) bool) AggregationOperator
```

Passes the value through only when `predicate` returns `true`. When `predicate` returns `false`, injects a pipeline error (`filter: value did not match predicate`). Skips when a pipeline error is already flowing.

Use `CatchError` downstream to handle the filtered-out case.

```go
agg.Pipe(
    Filter(func(c ginject.Context, data any) bool {
        return data != nil
    }),
    CatchError(func(c ginject.Context, err error) any {
        return map[string]any{"error": "no result"}
    }),
)
```

---

### `Take`

```go
func Take(n int) AggregationOperator
```

Passes the value through when `n > 0`. When `n <= 0`, injects a pipeline error (`take: count exceeded`). In a single-value pipeline this acts as a conditional gate: `Take(1)` always passes, `Take(0)` always errors.

Skips when a pipeline error is already flowing.

```go
agg.Pipe(
    Take(1), // pass through
)
```

---

### `ThrowError`

```go
func ThrowError(err error) AggregationOperator
```

Unconditionally injects `err` as a pipeline error, replacing any current value (including normal values and existing pipeline errors). The returned operator captures `err` at construction time; no allocation occurs at call time.

```go
agg.Pipe(
    ThrowError(ErrUnauthorized),
    CatchError(func(c ginject.Context, err error) any {
        return map[string]any{"error": err.Error()}
    }),
)
```

---

### `Delay`

```go
func Delay(d time.Duration) AggregationOperator
```

Pauses execution for `d` before passing the current value through. Applies to both normal values and pipeline errors (the delay is unconditional). Blocks the handler goroutine for the duration.

```go
agg.Pipe(
    Delay(200 * time.Millisecond),
    Map(fn),
)
```

---

### `Timeout` (method)

```go
func (aggregation *Aggregation) Timeout(d time.Duration) *Aggregation
```

Registers a timeout check that fires when `Aggregate` runs. The elapsed time is measured from **`c.Timestamp`** — the moment the request context was formed — not from the moment `Aggregate` is called. This means the budget covers the entire in-flight time of the request, including routing, middleware, and handler execution.

When `Aggregate` processes this operator:
- If `c` is `nil` or `c.Timestamp` is the zero value, the check is skipped (safe for tests and non-HTTP contexts).
- If `time.Since(c.Timestamp) >= d`, it panics with a `408 RequestTimeoutException`.
- Otherwise the pipeline value passes through unchanged.

Returns `*Aggregation` for method chaining.

```go
func (i TimeoutInterceptor) Intercept(c *ctx.Context, agg *aggregation.Aggregation) any {
    agg.Timeout(500 * time.Millisecond)
    return agg.Pipe(
        // other operators — only run if the budget has not been exceeded
    )
}
```

Chain multiple `Timeout` calls if you want different budgets at different pipeline stages:

```go
agg.Timeout(2 * time.Second)   // total request budget
```

**When to use:** Use `Timeout` to enforce an end-to-end request budget per interceptor. Unlike a goroutine-based approach, it does not spawn additional goroutines and has zero allocation cost on the happy path.

---

### `CombineLatest`

```go
func CombineLatest(others ...any) AggregationOperator
```

Combines the current pipeline value with additional static values into `[]any`. The result slice has the format:

```
[pipelineValue, others[0], others[1], ...]
```

Skips on pipeline error. Allocates a new slice per invocation.

```go
agg.Pipe(
    CombineLatest(requestID, timestamp),
    Map(func(c ginject.Context, data any) any {
        parts := data.([]any)
        return map[string]any{
            "result":    parts[0],
            "requestID": parts[1],
            "timestamp": parts[2],
        }
    }),
)
```

---

### `ForkJoin`

```go
func ForkJoin(fns ...func(*ctx.Context, any) any) AggregationOperator
```

Runs each function in `fns` concurrently, passing the current pipeline value to each. Waits for all goroutines to complete, then returns `[]any` with results in the same order as `fns`. Skips on pipeline error.

Each function writes to a dedicated index in a pre-allocated slice, so no mutex is needed.

```go
agg.Pipe(
    ForkJoin(
        func(c ginject.Context, data any) any { return fetchProfile(data.(string)) },
        func(c ginject.Context, data any) any { return fetchPermissions(data.(string)) },
    ),
    Map(func(c ginject.Context, data any) any {
        parts := data.([]any)
        return map[string]any{
            "profile":     parts[0],
            "permissions": parts[1],
        }
    }),
)
```

---

### `Scan`

```go
func Scan(fn func(*ctx.Context, any, any) any, seed any) AggregationOperator
```

Applies `fn(c, seed, currentValue)` and returns the result. `seed` is captured at operator construction and is constant across calls — it is not mutated. Skips on pipeline error.

In a single-value pipeline, `Scan` is equivalent to `Map` with a fixed second argument. Its primary use is readability when the transformation is an accumulation/fold:

```go
agg.Pipe(
    Scan(func(c ginject.Context, acc, data any) any {
        existing := acc.([]string)
        return append(existing, data.(string))
    }, []string{"prefix"}),
)
```

---

## Use cases

### 1. Wrapping every response in a standard envelope

```go
type ResponseInterceptor struct{}

func (i ResponseInterceptor) Intercept(c ginject.Context, agg ginject.Aggregation) any {
    return agg.Pipe(
        agg.Consume(func(c ginject.Context, data any) any {
            return map[string]any{"data": data, "success": true}
        }),
    )
}
```

### 2. Logging and metrics with `Tap` and `Finalize`

```go
func (i LoggingInterceptor) Intercept(c ginject.Context, agg ginject.Aggregation) any {
    start := time.Now()
    return agg.Pipe(
        Tap(func(c ginject.Context, data any) {
            log.Printf("handler returned %T in %s", data, time.Since(start))
        }),
        Finalize(func(c ginject.Context) {
            metrics.Histogram("request.duration", time.Since(start).Seconds())
        }),
    )
}
```

### 3. Error handling with `CatchError`

```go
func (i ErrorInterceptor) Intercept(c ginject.Context, agg ginject.Aggregation) any {
    return agg.Pipe(
        Map(func(c ginject.Context, data any) any {
            if data == nil {
                return ThrowError(ErrNotFound)(c, data) // inject error manually
            }
            return data
        }),
        CatchError(func(c ginject.Context, err error) any {
            return map[string]any{"error": err.Error()}
        }),
    )
}
```

### 4. Combining parallel data with `ForkJoin`

```go
func (i EnrichInterceptor) Intercept(c ginject.Context, agg ginject.Aggregation) any {
    return agg.Pipe(
        ForkJoin(
            func(c ginject.Context, data any) any {
                id := data.(User).ID
                return db.FetchRoles(id)
            },
            func(c ginject.Context, data any) any {
                id := data.(User).ID
                return db.FetchPreferences(id)
            },
        ),
        Map(func(c ginject.Context, data any) any {
            parts := data.([]any)
            user := /* original user from upstream interceptor */
            return EnrichedUser{Roles: parts[0], Prefs: parts[1]}
        }),
    )
}
```

### 5. Using the `Error` method for panic recovery

The `Error` method registers a handler that runs when a panic occurs anywhere in the request pipeline. It is independent of `CatchError` (which only handles pipeline errors from `ThrowError` / `Filter` / `Take`).

```go
func (i SafeInterceptor) Intercept(c ginject.Context, agg ginject.Aggregation) any {
    agg.Error(func(c ginject.Context, data any) any {
        // data is the panic value (string, error, etc.)
        log.Printf("panic recovered: %v", data)
        return map[string]any{"error": "internal server error"}
    })
    return agg.Pipe(
        agg.Consume(func(c ginject.Context, data any) any {
            return map[string]any{"data": data}
        }),
    )
}
```

### 6. End-to-end request timeout

```go
// Enforce a 500ms total budget counted from when the request arrived.
// If routing + handler + aggregation together exceed 500ms, the interceptor
// panics with 408 before writing the response.
func (i TimeoutInterceptor) Intercept(c *ctx.Context, agg *aggregation.Aggregation) any {
    agg.Timeout(500 * time.Millisecond)
    return agg.Pipe(
        agg.Consume(func(c *ctx.Context, data any) any {
            return map[string]any{"data": data}
        }),
    )
}
```

The 408 panic is caught by the framework's error handler chain (registered via `agg.Error`):

```go
agg.Error(func(c *ctx.Context, data any) any {
    if ex, ok := data.(exception.Exception); ok && ex.GetCode() == "408" {
        return map[string]any{"error": "request timed out"}
    }
    return map[string]any{"error": "internal server error"}
})
```
