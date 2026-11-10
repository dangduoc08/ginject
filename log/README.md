# log

`log` is ginject's default logger. It is built on `log/slog` and implements `common.Logger`:

```go
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
}
```

It gives you three things on top of `slog`:
- a **pretty tree** format for development;
- **masking** of sensitive fields (`MaskFields`);
- **struct tags** that rename or hide fields (`log:"name"`, `log:"-"`).

## What is supported

Every guarantee below holds in all three formats (Pretty, Text and JSON) and is covered by tests or fuzz tests:

| Input | Masked | Pretty tree |
|---|---|---|
| key/value pairs (`"k", v`) and `slog.Attr` | yes | yes |
| `slog.Group` | yes | yes, as prefixed keys |
| `slog.Logger.With`, `WithGroup` (only when you build your own `slog.Logger`; `common.Logger` has neither) | **no**: these attributes reach the handler directly, without masking | yes, as prefixed keys |
| maps, slices and arrays, at any depth | yes | yes |
| structs (exported fields, `log` tags) | yes | yes |
| `slog.LogValuer` and `slog.Value`, at any depth | yes, resolved first | yes, resolved first |
| `slog.Attr` and `[]slog.Attr` nested as values | yes, like a group | yes, as a map |
| `error`, `fmt.Stringer` | only as a whole, by key; the text itself is not inspected | yes, as text |

Anything not in this table, such as unexported fields or values hidden behind other interfaces, is not part of the contract.

## Usage

```go
logger := log.NewLog(&log.LogOptions{
	Level:      log.DebugLevel,
	LogFormat:  log.PrettyFormat,
	MaskFields: []string{"password", "card.cvv"},
})

app := core.New()
app.UseLogger(logger)
```

If you don't call `UseLogger`, `app.Create` calls `log.NewLog(nil)`, which gives a pretty logger at Info level with no masking.

Arguments follow `slog`: alternating key/value pairs, `slog.Attr` values, or a mix of both.

```go
logger.Info("UserCreated", "user", user, "tags", []string{"new", "vip"}, "error", err)
logger.Info("UserCreated", slog.Int("id", 42), slog.Group("http", slog.String("method", "GET")))
```

`Fatal` logs at `FatalLevel` and then calls `os.Exit(1)`. Deferred functions don't run.

## Options

| Field | Default | Meaning |
|---|---|---|
| `Level` | `InfoLevel` (zero value) | Minimum level written. Any `slog.Level` works, including custom levels between the named ones |
| `LogFormat` | `PrettyFormat` | `PrettyFormat`, `TextFormat` (`slog.TextHandler`) or `JSONFormat` (`slog.JSONHandler`) |
| `TimeFormat` | `time.DateTime` | Timestamp layout. Used by `PrettyFormat` only |
| `AddSource` | `false` | Adds the caller's `file:line` |
| `MaskFields` | none | Patterns of keys to redact. See [Masking](#masking) |

Levels: `DebugLevel`, `InfoLevel`, `WarnLevel`, `ErrorLevel`, `FatalLevel` (`slog.Level(12)`).

Output always goes to `os.Stdout`.

`PrettyFormat` is the default and prints ANSI colours whether or not stdout is a terminal. In production, where a log collector reads the output (Docker, Kubernetes, systemd), set `LogFormat: log.JSONFormat` yourself.

## Pretty format

```
 INFO   2026-11-10 22:48:44   [UserCreated]
  ├── user
  │   ├── Card
  │   │   ├── cvv    [REDACTED]
  │   │   └── number 4111
  │   ├── Email      a@b.co
  │   ├── Password   [REDACTED]
  │   └── id         42
  ├── tags
  │   ├── new
  │   └── vip
  └── error          smtp timeout
```

The example was logged with `MaskFields: []string{"password", "card.cvv"}`.

- **Layout:** values are aligned across the whole record. Maps, slices and structs become branches, and map keys are sorted.
- **Colours:** keys are red, strings green, numbers orange and booleans blue. An `error` prints its `Error()` in red, and any `fmt.Stringer` (`time.Time`, `time.Duration`, …) prints its `String()`. Values implementing `slog.LogValuer` are resolved first.
- **Coloured values:** a `log.Colored{Text, Color}` value prints its text in its own colour. The access log uses this for stage durations.
- **Grouping:** `slog.Group(...)` attributes, `slog.Logger.With(...)` and `WithGroup(name)` keep the pretty format. Grouped keys are printed with a prefix, `http.method`, like `slog.TextHandler` does. A group with an empty key is inlined, and an empty group is dropped.

### Escaping

Control bytes (below `0x20`, and `0x7f`) in the message, keys and string values are printed escaped: `\n`, `\r`, `\t`, `\x1b`. Text that comes from a client therefore can't forge extra log lines or send terminal escape codes:

```
 WARN   2026-11-10 22:48:44   [GET /a\n[FATAL] fake]
  └── ua curl\x1b[2J
```

### Limits

Each record stops at a depth of **16** and at **1000** nodes.
- A cyclic value, such as a map that contains itself or a struct that points back to its parent, prints `[MAX_DEPTH]` where the depth runs out.
- Children beyond the node limit are summarised as `… (N more)`. When a map is cut, the keys shown are a subset in iteration order, sorted, so which keys appear can change from one call to the next.

## Masking

`MaskFields` takes patterns in the same syntax as the `pattern` package (see `pattern/MATCHING.md`):
- `*` matches exactly one segment;
- a trailing `**` matches one or more segments.

The path of a value is its keys joined by dots: `user.card.cvv`.

**The rule:**
- Every pattern is matched against every **suffix** of the path, at every level.
- Matching is **case-insensitive**.
- A value whose path matches is replaced **whole** by `[REDACTED]`, even if it is a map or a struct.

| Pattern | Logged | Result |
|---|---|---|
| `password` | `"password", "p"` | `password: [REDACTED]` |
| `password` | `"user", map{"password": "p"}` | `user.password: [REDACTED]` |
| `password` | `"user", User{Password: "p"}` | `Password: [REDACTED]` (case-insensitive) |
| `password` | `"users", []User{…}` | `Password` masked in every element |
| `password` | `slog.String("password", "p")` | masked |
| `password` | `"userpassword", "p"` | not masked: a match needs a whole segment |
| `card` | `"user", map{"card": map{"cvv": "1"}}` | `card: [REDACTED]`, the whole map |
| `card.*` | `"user", map{"card": map{"cvv": "1"}}` | `user.card.cvv: [REDACTED]`, through the suffix `card.cvv` |
| `user.*` | `"user", map{"card": …, "name": "a"}` | `card` and `name` both masked whole |
| `user.password` | `"admin", map{"password": "p"}` | not masked: no suffix is `user.password` |

**Details:**
- Elements of a slice share the slice's path: `users` → `users.password`.
- Inside a `slog.Group`, keys are joined to the group's key. A group with an empty key adds nothing to the path.
- A value implementing `slog.LogValuer`, or a `slog.Value`, is resolved first and its result is masked, so `LogValue()` can't leak a masked field. When nothing in it matches, the value is passed on unchanged.
- Masking only runs when the level is enabled, so a filtered `Debug` call costs almost nothing.

`WrapLogger(next, maskFields)` adds the same masking to any `common.Logger`, including your own. `NewLog` already calls it.

## Struct tags

```go
type User struct {
	ID       int    `log:"id"`
	Email    string
	Password string `log:"-"`
	token    string
}
```

| Field | Logged as |
|---|---|
| `ID` | `id` |
| `Email` | `Email`, the Go name |
| `Password` | never logged |
| `token` | never logged. Unexported fields are always skipped, with or without a tag |

Masking and the pretty format both use the tag names, so `MaskFields: []string{"id"}` masks `ID`.

Through `WrapLogger`, a struct is turned into a map, so in the pretty format its fields are printed in sorted key order rather than declaration order.

Types that implement `error`, `fmt.Stringer` or `slog.LogValuer` are never expanded into fields.

## Concurrency and performance

- **Safe for concurrent use.** Many goroutines can log at once. Records never interleave, and none are lost (tested with 16 goroutines × 200 records under `-race`).
- **Formatting happens outside the lock.** The pretty handler formats each record in a pooled buffer, then takes a lock only to write it. At 12 CPUs this costs about 700–1000 ns per record, against about 1.9 µs in master.
- **Writes are synchronous**, as in `slog`, `zap` and `zerolog`. If stdout blocks (a stalled pipe, a paused terminal, a blocked log driver), every goroutine that logs waits for it. An opt-in async writer is planned (ROADMAP R-4).
- **`NewLog` is a singleton.** The first call decides the options. Later calls return the same logger and ignore their arguments.

Rough costs (Go 1.27.3, Intel i7-9750H):

| Call | Cost |
|---|---|
| `Debug` filtered out by level | ~57 ns |
| `Info` with scalar args, no masking rules | ~51 ns before the handler |
| Pretty `Handle`, 3 scalar attrs | ~700 ns |
| Pretty `Handle`, access-log record with nested stages | ~7 µs |
