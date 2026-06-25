# DS Package

*`ds` is an internal package that implements the segment-based trie Ginject's router uses to register routes and match request paths against them.*

- [DS Package](#ds-package)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [`Node` Type](#node-type)
  - [`Trie` Struct](#trie-struct)
    - [Index](#index)
    - [Raw](#raw)
    - [Children](#children)
  - [Functions](#functions)
    - [NewTrie](#newtrie)
  - [`*Trie` Methods](#trie-methods)
    - [Len](#len)
    - [Insert](#insert)
    - [Find](#find)
    - [ToJSON](#tojson)
  - [Benchmarks](#benchmarks)

## Key Features
- Segments paths on a caller-supplied separator byte instead of assuming `/`
- Three segment kinds: literal text, `$` for a captured parameter, `*` for a wildcard
- `Find` reports an exact match and the best wildcard fallback in a single pass
- `ToJSON` dumps the trie shape for debugging or visualization

## Usage

A `Trie` stores a path under two strings: the `raw` string (returned later on a match) and the string that is actually walked to build the trie, where `$` marks a dynamic segment and `*` marks a wildcard segment. Both inserts and lookups must use the same separator byte:

```go
package main

import (
	"fmt"

	"github.com/dangduoc08/ginject/internal/ds"
)

func main() {
	tr := ds.NewTrie()

	tr.Insert("/users/:id/", "/users/$/", '/', 0)
	tr.Insert("/users/:id/friends/", "/users/$/friends/", '/', 1)

	index, raw, wildcardIndex, wildcardRaw, params := tr.Find("/users/123/", '/')

	fmt.Println("matched index:", index)
	fmt.Println("matched raw:", raw)
	fmt.Println("wildcard index:", wildcardIndex)
	fmt.Println("wildcard raw:", wildcardRaw)
	fmt.Println("params:", params)
}
```

Console:
```console
matched index: 0
matched raw: /users/:id/
wildcard index: -1
wildcard raw:
params: [123]
```

## `Node` Type
Type: `map[string]*Trie`

`Node` is the map type behind `Trie.Children`. Each key is a single path segment — a literal token, or one of the special tokens `$` (captured parameter) and `*` (wildcard) — mapped to the child `Trie` reached by that segment.

## `Trie` Struct

### Index
Type: `int`

Default: `-1`

Required: `false`

The identifier stored on the node that terminates an inserted path. `NewTrie` initializes it to `-1`, which `Find` treats as "no route registered here." `Insert` only overwrites it (with its `index` argument) on the node matching the last segment of the inserted string.

### Raw
Type: `string`

Default: `""`

Required: `false`

The raw route string passed as `Insert`'s first argument, stored on the node that terminates that path. `Find` returns it as `matchedRaw`/`wildcardRaw` so the original pattern can be recovered after a match.

### Children
Type: `Node`

Default: an empty, non-nil map (`make(Node)`, set by `NewTrie`)

Required: `false`

The node's child segments, keyed by literal text or by the special `$`/`*` tokens.

## Functions

### NewTrie

Creates an empty trie ready for `Insert` and `Find`.

#### Rules
- Returns a trie with `Index` set to `-1` and an empty, non-nil `Children` map; calling `Len()` immediately after returns `0` (`TestTrieLenEmpty`).

#### Parameters
None.

#### Returns
- 1st value: `*Trie`

- Description: A new trie with `Index` set to `-1` and an empty `Children` map.

#### Usage

```go
tr := ds.NewTrie()
```

## `*Trie` Methods

### Len

Counts every node in the trie below the receiver, i.e. one per inserted path segment across all routes (not just leaves).

#### Rules
- An empty trie's `Len()` is `0` (`TestTrieLenEmpty`).
- `Len()` counts one node per distinct segment across all inserted paths; segments shared by multiple routes (a common prefix) are counted once, not once per route — three routes sharing prefixes produce `Len() == 6`, not `3` (`TestTrieLen`).

#### Parameters
None.

#### Returns
- 1st value: `int`

- Description: Total number of descendant nodes.

#### Usage

```go
tr := ds.NewTrie()
tr.Insert("/users/{userId}/", "/users/{userId}/", '/', -1)
tr.Insert("/feeds/all/", "/feeds/all/", '/', -1)
tr.Insert("/users/{userId}/friends/all/", "/users/{userId}/friends/all/", '/', -1)

fmt.Println(tr.Len())
```

Console:
```console
6
```

### Insert

Splits `insertedStr` on `sep` and walks/creates a child node per segment, storing `raw` and `index` on the node for the final segment. Use the literal segment `$` to mark a dynamic (parameter) segment and `*` to mark a wildcard segment. Returns the receiver, so calls can be chained.

#### Rules
- Only the node for the final segment of `insertedStr` receives the supplied `index` and `raw`; every intermediate segment node keeps its default `Index` of `-1` unless that same segment is also the terminal segment of a different inserted path (`TestTrieInsert`).
- Inserting paths that share a prefix reuses the existing nodes for that prefix instead of creating duplicates (`TestTrieInsert`, `TestTrieLen`).

#### Parameters
- 1st parameter: `string` (`raw`)

- Description: The original route string to store on the matched node; returned later by `Find`.

- 2nd parameter: `string` (`insertedStr`)

- Description: The string that is actually segmented and walked to build the trie path. Use `$` and `*` segments for parameters and wildcards.

- 3rd parameter: `byte` (`sep`)

- Description: The separator byte used to split `insertedStr` into segments.

- 4th parameter: `int` (`index`)

- Description: An identifier to store on the node for the final segment; returned later by `Find`.

#### Returns
- 1st value: `*Trie`

- Description: The receiver trie, returned to allow chaining further `Insert` calls.

#### Usage

```go
tr := ds.NewTrie()
tr.
	Insert("/users/:id/", "/users/$/", '/', 0).
	Insert("/feeds/all/", "/feeds/all/", '/', 1)
```

### Find

Walks `path` segment by segment on `sep`, preferring an exact literal match at each level, then a `$` (param) child, then a `*` (wildcard) child, falling back to comparing against any sibling segment containing a literal `*` pattern (e.g. `*.html`). While traversing, it also tracks the most specific `*` child passed through so a wildcard fallback is available even when no exact match is found.

#### Rules
- A path must be consumed exactly to a terminal node to count as a match: a path that is an incomplete prefix of a registered route returns `""` for both `matchedRaw` and `wildcardRaw` (`TestTrieFind`, "incomplete path should not match").
- `$` segments capture their literal path value into `paramVals`, in left-to-right traversal order (`TestTrieFind`, "deep param match").
- Once the path passes through a `*` node, that node's `Index`/`Raw` are reported via `wildcardIndex`/`wildcardRaw`, and the match still holds even when the path has extra trailing segments beyond the wildcard route's own length (`TestTrieFind`, "wildcard deep match, extra trailing segments").
- A wildcard match is used as a fallback even when an unrelated, deeper sibling route exists on a different branch that doesn't match the path (`TestTrieFindWildcardFallbackThroughUnrelatedSibling`).

#### Parameters
- 1st parameter: `string` (`path`)

- Description: The path to look up, using the same separator that was used at insert time.

- 2nd parameter: `byte` (`sep`)

- Description: The separator byte used to split `path` into segments.

#### Returns
- 1st value: `int`

- Description: `Index` of the node that exactly matches the full path, or `-1` if there is no exact match.

- 2nd value: `string`

- Description: `Raw` of that exactly matched node, or `""` if there is no exact match.

- 3rd value: `int`

- Description: `Index` of the most specific wildcard (`*`) node encountered while traversing the path, or `-1` if none was passed through.

- 4th value: `string`

- Description: `Raw` of that wildcard node, or `""` if none was passed through.

- 5th value: `[]string`

- Description: Values captured for each `$` segment, in the order they were matched.

#### Usage

```go
tr := ds.NewTrie()
tr.Insert("/users/:id/", "/users/$/", '/', 0)

index, raw, wildcardIndex, wildcardRaw, params := tr.Find("/users/123/", '/')
fmt.Println(index, raw, wildcardIndex, wildcardRaw, params)
```

Console:
```console
0 /users/:id/ -1  [123]
```

### ToJSON

Serializes the trie's shape — every segment's path, `Index`, and children — to a JSON string. Because `Children` is a Go map, the order of sibling entries in the output is not guaranteed to be stable between calls (the keys within each JSON object are always `children`, `index`, `path`, sorted alphabetically by `encoding/json`).

#### Rules
- The root node's JSON object has no `"path"` key, only `"children"`; every other node includes `"path"` (its segment key), `"index"`, and `"children"` (`TestTrieToJSON`).

#### Parameters
None.

#### Returns
- 1st value: `string`

- Description: JSON representation of the trie.

- 2nd value: `error`

- Description: Non-nil if JSON marshaling fails.

#### Usage

```go
tr := ds.NewTrie()
tr.Insert("/users/$/", "/users/$/", '/', 0)
tr.Insert("/feeds/all/", "/feeds/all/", '/', 1)

js, err := tr.ToJSON()
if err != nil {
	panic(err)
}
fmt.Println(js)
```

Console (one possible ordering — sibling order may vary):
```console
{"children":[{"children":[{"children":[],"index":0,"path":"$"}],"index":-1,"path":"users"},{"children":[{"children":[],"index":1,"path":"all"}],"index":-1,"path":"feeds"}]}
```

## Benchmarks

Captured by running `go test -run=^$ -bench=. -benchmem ./internal/ds/...`. Numbers are machine-dependent and were captured at doc-generation time — re-run the command yourself for a fresh baseline.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/internal/ds
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkMatchWildcard-12         	56382829	        22.31 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_Static-12       	18563512	        64.83 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_WithParam-12    	 7714236	       149.9 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_DeepParam-12    	 4847652	       256.6 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_NoMatch-12      	 6658142	       179.4 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/internal/ds	7.329s
```
