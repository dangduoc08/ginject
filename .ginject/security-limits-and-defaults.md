# Security Limits & Defaults

Every resource cap and security default in the framework, what it closes, and how to opt out.

**Read this before changing any default, removing any cap, or "simplifying" any guard below.** Each row exists because the absence was exploitable. Confidence: HIGH — every entry was reproduced before the fix and re-verified after.

Cross-refs: [anti-patterns-gotchas.md](anti-patterns-gotchas.md) · [request-pipeline.md](request-pipeline.md) · [websocket-system.md](websocket-system.md) · [metadata-concurrency-model.json](metadata-concurrency-model.json)

---

## 1. Caps truth table

| Cap | Constant / field | Default | Enforced at | Opt out |
|---|---|---|---|---|
| HTTP request body | `core.DefaultMaxRequestBodyBytes` | 10 MB | `App.ServeHTTP` wraps `r.Body` in `http.MaxBytesReader` | `app.SetMaxRequestBodySize(0)` |
| WS connections | `WSConfig.MaxConnections` / `core.DefaultWSMaxConnections` | 10000 | `WSConnmgr.Register` (check + insert under one write lock) | set to 0 |
| WS frame payload | `WSConfig.MaxPayloadBytes` / `core.DefaultWSMaxPayloadBytes` | 1 MB | `wsConn.MaxPayloadBytes` in `handleRequest` | set higher |
| WS write stall | `WSConfig.WriteTimeout` / `core.DefaultWSWriteTimeout` | 10 s | `SetWriteDeadline` before every `websocket.JSON.Send` | set to 0 |
| Cache entries | `memorycache.DefaultMaxEntries` | 100 000 | `shard.admitLocked` on every write path | `WithMaxEntries(0)` |
| HTTP header | `MaxHeaderBytes` | 1 MB | `http.Server` in `Listen` | edit `Listen` |
| Multipart in-memory | `ctx.defaultMaxMemory` | 32 MB | `ParseMultipartForm` | — |

`0` or negative disables a cap everywhere it is accepted. Below 10 MB body cap, multipart never spills to disk in the default config (10 MB < 32 MB in-memory limit).

---

## 2. What each cap closes

| Cap absent | Exploit |
|---|---|
| body cap | `io.ReadAll(r.Body)` in `ctx.Body()` is unbounded. `ReadTimeout 5s` bounds *time*, not *size* — a fast client pushes hundreds of MB in 5 s → OOM. `MaxHeaderBytes` does **not** help; it bounds headers only. |
| WS connection cap | 3 goroutines + a 32-slot buffer per connection, unbounded → goroutine/memory exhaustion. |
| WS payload cap | `x/net/websocket` defaults to `DefaultMaxPayloadBytes` = **32 MB** (not unbounded). 32 MB × N connections is still a memory DoS. |
| WS write deadline | A peer that never reads blocks `websocket.JSON.Send` forever. `close(done)` **cannot** free it — the goroutine is blocked inside `Send`, not in its `select`. Permanent goroutine leak per stalled client. |
| cache cap | Keys with no TTL live forever; a workload keyed off request data grows memory without limit. |

---

## 3. Security defaults that reject configuration

### CORS: wildcard + credentials is a config error

`loadCORSOptions` **panics** when `AllowOrigin` resolves to `"*"` and `IsAllowCredentials` is true. This includes leaving `AllowOrigin` unset — `normalizeAllowOrigin(nil)` returns `"*"`.

```go
cors.CORS{IsAllowCredentials: true}                  // PANICS at config time
cors.CORS{AllowOrigin: "*", IsAllowCredentials: true} // PANICS
cors.CORS{AllowOrigin: "*"}                           // fine, emits literal *
cors.CORS{AllowOrigin: []string{"https://a.example"}, IsAllowCredentials: true} // correct
```

Why fail instead of echo: browsers reject `*` on credentialed requests *by design*. Echoing the request origin back to satisfy them converts that protection into a full bypass — every site on the internet gains credentialed read access to authenticated responses. There is no safe interpretation, so it fails fast at startup rather than silently.

Panics at `NewMiddleware()` (bootstrap). The uncompiled `CORS{}.Use(...)` path calls `loadCORSOptions` per request; the framework does not use it.

### Throttler: proxy headers are not trusted

`Throttler.KeyFunc` defaults to `remoteAddrThrottlerKeyFunc` — `RemoteAddr` only.

`X-Real-IP` and `X-Forwarded-For` are client-supplied. Honouring them by default lets any caller send a fresh random header per request, getting a fresh bucket each time and bypassing the limit **entirely**, while growing the counter store without bound.

`TrustProxyHeaders: true` switches to `proxyAwareThrottlerKeyFunc` (`X-Real-IP`, then first `X-Forwarded-For` entry, then `RemoteAddr`). Set it **only** when a trusted proxy overwrites those headers.

### CSRF cookie attributes

| Attribute | Value | Why |
|---|---|---|
| `SameSite` | `Lax` default | Keeps the cookie off cross-site POSTs. `CSRF.SameSite` overrides. |
| `Secure` | auto when `r.TLS != nil` | Does not break plain-HTTP dev. `CSRF.Secure: true` forces it for TLS-terminating proxies. |
| `HttpOnly` | **false**, deliberately | Double-submit requires JS to read the cookie. Do not "harden" this to true — it breaks the scheme. |
| `Path` | `/` | `CSRF.CookiePath` overrides. |

**Trap:** the zero value of `http.SameSite` is `0`, which is *not* `SameSiteDefaultMode` (= 1). Compare against `0` when defaulting, or `SetCookie` emits no attribute at all.

### WS origin

`WSConfig.AllowedOrigins`, when non-empty, is enforced in `ws.handshake` before the upgrade completes; an unlisted origin returns `errWSOriginRejected`. An **absent** `Origin` header is allowed — it marks a non-browser client, which the header cannot protect against anyway. Trailing `/` is trimmed on both sides.

When `AllowedOrigins` is empty **and** no middleware was passed to `EnableWS`, `NewWS` logs `WSOriginUnrestricted`.

Origin checking is **also** available through the CORS middleware: `compiledCORS.Use` detects `Upgrade: websocket` and calls `next()` only when the origin matches. So passing a configured `cors.CORS` to `EnableWS` gates WS origins too. The gap the framework had was the *default*: a custom `websocket.Server.Handshake` replaces `x/net`'s built-in same-origin check, so with no CORS middleware and no `AllowedOrigins` there was no check at all — less safe than the library underneath.

---

## 4. Log masking

`log.WrapLogger(next, maskFields)` masks by **key path**, walking structs, string-keyed maps, pointers, **slices and arrays**.

| Shape | Walked |
|---|---|
| struct, pointer-to-struct | yes |
| `map[string]...` | yes |
| slice / array of struct, map, interface, pointer, slice, array | yes — element path = the collection's path |
| `[]byte`, `[]string`, other scalar slices | **no**, left as the concrete type |

Collections were previously not walked, so `logger.Info("users", users)` leaked every masked field in the collection.

`elemCanHoldSecrets` is what keeps `[]byte` from being exploded into a list of numbers. Do not drop it.

**Matching is case-sensitive.** An untagged Go field is keyed by its exported name, so rule `password` does **not** reach field `Password`. Give fields a `log:"password"` tag, or write the rule in matching case.

---

## 5. Breaking changes vs. pre-hardening behaviour

| Change | Previous | Restore old behaviour |
|---|---|---|
| request body capped | unlimited | `app.SetMaxRequestBodySize(0)` |
| WS payload capped | 32 MB (`x/net` default) | `WSConfig.MaxPayloadBytes` |
| WS connections capped | unlimited | `MaxConnections: 0` |
| cache entries capped | unlimited | `WithMaxEntries(0)` |
| throttler ignores proxy headers | trusted them | `TrustProxyHeaders: true` |
| CORS wildcard+credentials panics | echoed request origin | enumerate origins (no opt-out — the old behaviour was the vulnerability) |
| `Devtool.Serve` signature | `Serve()` | `Serve(ctx) error` |
| `WSConnmgr.Register` | always returned a conn | may return `nil`; callers must nil-check |

---

## 6. Forbidden

- Never remove a cap to "fix" a large-payload bug. Raise the cap explicitly at the call site.
- Never set `CORS.IsAllowCredentials` with a wildcard origin. It panics; do not work around the panic.
- Never enable `TrustProxyHeaders` unless a proxy actually overwrites those headers.
- Never set CSRF `HttpOnly: true` — it breaks double-submit.
- Never hold a lock across a user callback (`common.Construct` does; see [anti-patterns-gotchas.md](anti-patterns-gotchas.md)).
- Never assume `MultipartForm` temp files leak: `net/http`'s `finishRequest` calls `RemoveAll()` after the handler returns (verified in the stdlib source, both HTTP/1 and h2).
