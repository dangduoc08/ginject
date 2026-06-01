# matcher

Standalone topic pattern matching for dot-separated event topics. No external dependencies, no broker dependency — can be imported by any adapter or transport layer.

---

## Public API

```go
import "github.com/dangduoc08/ginject/matcher"

// Parse converts a raw pattern string into a Pattern value.
// Parsing happens once at subscribe time; matching is then allocation-free
// for Exact, Global, and SingleSuffix kinds.
func Parse(raw string) Pattern

// Match reports whether topic satisfies pattern p.
func Match(p Pattern, topic string) bool
```

### Pattern type

```go
type Pattern struct { /* unexported */ }

func (p Pattern) Raw() string          // original pattern string
func (p Pattern) Kind() Kind           // one of the four kinds below
func (p Pattern) SimplePrefix() string // non-empty only for KindSingleSuffix
func (p Pattern) IsExact() bool
func (p Pattern) IsGlobal() bool
```

### Kind constants

```go
type Kind uint8

const (
    KindExact        Kind = iota // no wildcards
    KindGlobal                   // "*" or ">"
    KindSingleSuffix              // "prefix.*"
    KindComplex                   // everything else
)
```

---

## Pattern reference

| Pattern | Kind | Matches | Does not match |
|---|---|---|---|
| `user.created` | `KindExact` | `user.created` | `user.updated` |
| `*` | `KindGlobal` | every topic | — |
| `>` | `KindGlobal` | every topic | — |
| `user.*` | `KindSingleSuffix` | `user.created`, `user.deleted` | `user.profile.updated` |
| `a.b.*` | `KindSingleSuffix` | `a.b.c` | `a.b.c.d` |
| `user.>` | `KindComplex` | `user.created`, `user.profile.updated`, `user.a.b.c` | `user` |
| `tenant.*.user.created` | `KindComplex` | `tenant.1.user.created`, `tenant.abc.user.created` | `tenant.1.user.updated` |
| `tenant.*.user.>` | `KindComplex` | `tenant.1.user.created`, `tenant.1.user.profile.updated` | `tenant.1.admin.x` |
| `*.created` | `KindComplex` | `user.created`, `order.created` | `a.b.created` |

### Wildcard semantics

- `*` — matches exactly **one** segment when used inside a pattern (`user.*.created`) or the **entire topic** when used alone (alias for `>`).
- `>` — matches **one or more** remaining segments. Must be the last token. A topic must have at least one segment after the preceding literal.
- Patterns are **dot-separated**. Segments are the substrings between dots.

---

## Usage

```go
p := matcher.Parse("user.*")
matcher.Match(p, "user.created")        // true
matcher.Match(p, "user.profile.updated") // false — two levels deep

p2 := matcher.Parse("tenant.*.user.>")
matcher.Match(p2, "tenant.1.user.created")          // true
matcher.Match(p2, "tenant.abc.user.profile.updated") // true
matcher.Match(p2, "tenant.1.admin.created")          // false

// Check kind at subscribe time to route to the right bucket:
switch p.Kind() {
case matcher.KindExact:
    // O(1) map lookup
case matcher.KindGlobal:
    // matches everything
case matcher.KindSingleSuffix:
    prefix := p.SimplePrefix() // "user" for "user.*"
    // O(1) lookup via lastDot(topic)
case matcher.KindComplex:
    // O(depth) segment scan
}
```

---

## Performance

Patterns are parsed **once** at subscribe time. Matching costs:

| Kind | Match cost | Allocs |
|---|---|---|
| `KindExact` | O(1) string compare | 0 |
| `KindGlobal` | O(1) return true | 0 |
| `KindSingleSuffix` | O(prefix length) | 0 |
| `KindComplex` | O(topic depth) | 1 (`strings.Split`) |

Benchmark results (Intel Core i7-9750H):

| Benchmark | ns/op | allocs/op |
|---|---|---|
| `BenchmarkMatch_Exact` | ~5 | 0 |
| `BenchmarkMatch_Global` | ~4 | 0 |
| `BenchmarkMatch_SingleSuffix_Hit` | ~28 | 0 |
| `BenchmarkMatch_Complex_3Seg` | ~117 | 1 |
| `BenchmarkMatch_Complex_Multi` | ~98 | 1 |
| `BenchmarkParse_Exact` | ~72 | 1 |
| `BenchmarkParse_Complex` | ~81 | 1 |
