# DS Package

*`ds` is an internal package that implements the segment-based trie Ginject's router uses to register routes and match request paths against them.*

- [DS Package](#ds-package)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [`Node` Type](#node-type)
  - [`Trie` Struct](#trie-struct)
    - [IsEnd](#isend)
    - [Raw](#raw)
    - [Children](#children)
  - [Functions](#functions)
    - [NewTrie](#newtrie)
  - [`*Trie` Methods](#trie-methods)
    - [Len](#len)
    - [Insert](#insert)
    - [Remove](#remove)
    - [Find](#find)
    - [ToJSON](#tojson)
  - [Benchmarks](#benchmarks)

## Key Features
- Segments paths on a caller-supplied separator byte instead of assuming `/`
- Three segment kinds: literal text, `$` for a captured parameter, `*` for a wildcard
- `$`-param matching is opt-in per `Find` call: passing `false` skips the `$`-child lookup entirely (the common case for packages like `broker`/`broker1` that never register `$` segments); passing `true` enables it
- `Find` reports an exact match and the best wildcard fallback in a single pass
- `Remove` un-registers a previously inserted path and prunes every dead ancestor node it leaves behind
- `ToJSON` dumps the trie shape for debugging or visualization

## Usage

A `Trie` stores a path under two strings: the `raw` string (returned later on a match) and the string that is actually walked to build the trie, where `$` marks a dynamic segment and `*` marks a wildcard segment. Both inserts and lookups must use the same separator byte. Matching `$` segments as captured parameters requires passing `true` as `Find`'s third argument — passing `false` treats `$` as an ordinary literal segment:

```go
package main

import (
	"fmt"

	"github.com/dangduoc08/ginject/internal/ds"
)

func main() {
	tr := ds.NewTrie()

	tr.Insert("/users/:id/", "/users/$/", '/')
	tr.Insert("/users/:id/friends/", "/users/$/friends/", '/')

	raw, wildcardRaw, params := tr.Find("/users/123/", '/', true)

	fmt.Println("matched raw:", raw)
	fmt.Println("wildcard raw:", wildcardRaw)
	fmt.Println("params:", params)
}
```

Console:
```console
matched raw: /users/:id/
wildcard raw:
params: [123]
```

## `Node` Type
Type: `map[string]*Trie`

`Node` is the map type behind `Trie.Children`. Each key is a single path segment — a literal token, or one of the special tokens `$` (captured parameter) and `*` (wildcard) — mapped to the child `Trie` reached by that segment.

## `Trie` Struct

### IsEnd
Type: `bool`

Default: `false`

Required: `false`

Marks the node that terminates an inserted path. `NewTrie` leaves it at its zero value `false`, which `Find` treats as "no route registered here." `Insert` only sets it to `true` on the node matching the last segment of the inserted string; `Remove` resets it to `false`.

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
- Returns a trie with `IsEnd` set to `false` and an empty, non-nil `Children` map; calling `Len()` immediately after returns `0` (`TestTrieLenEmpty`).

#### Parameters
None.

#### Returns
- 1st value: `*Trie`

- Description: A new trie with `IsEnd` set to `false` and an empty `Children` map.

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
tr.Insert("/users/{userId}/", "/users/{userId}/", '/')
tr.Insert("/feeds/all/", "/feeds/all/", '/')
tr.Insert("/users/{userId}/friends/all/", "/users/{userId}/friends/all/", '/')

fmt.Println(tr.Len())
```

Console:
```console
6
```

### Insert

Splits `insertedStr` on `sep` and walks/creates a child node per segment, storing `raw` and marking `IsEnd` on the node for the final segment. Use the literal segment `$` to mark a dynamic (parameter) segment and `*` to mark a wildcard segment. Returns the receiver, so calls can be chained.

#### Rules
- Only the node for the final segment of `insertedStr` has `IsEnd` set to `true` and receives `raw`; every intermediate segment node keeps its default `IsEnd` of `false` unless that same segment is also the terminal segment of a different inserted path (`TestTrieInsert`).
- Inserting paths that share a prefix reuses the existing nodes for that prefix instead of creating duplicates (`TestTrieInsert`, `TestTrieLen`).

#### Parameters
- 1st parameter: `string` (`raw`)

- Description: The original route string to store on the matched node; returned later by `Find`.

- 2nd parameter: `string` (`insertedStr`)

- Description: The string that is actually segmented and walked to build the trie path. Use `$` and `*` segments for parameters and wildcards.

- 3rd parameter: `byte` (`sep`)

- Description: The separator byte used to split `insertedStr` into segments.

#### Returns
- 1st value: `*Trie`

- Description: The receiver trie, returned to allow chaining further `Insert` calls.

#### Usage

```go
tr := ds.NewTrie()
tr.
	Insert("/users/:id/", "/users/$/", '/').
	Insert("/feeds/all/", "/feeds/all/", '/')
```

### Remove

Walks `removedStr` segment by segment on `sep`, following only existing children (never creating nodes). If the path doesn't lead to a node previously terminated by `Insert` (i.e. one with `IsEnd == true`), the trie is left untouched and `Remove` returns `false`. Otherwise it clears that node's `IsEnd`/`Raw`, then walks back up the same path deleting every ancestor node that is now both childless and not itself an end node, stopping at the first ancestor that still holds a child or is itself a registered `IsEnd` node.

Like `Insert` and `Find`, `Remove` has no internal lock — it mutates the same `Children` maps `Insert` writes and `Find` reads, so a caller running `Remove` concurrently with `Insert`/`Find` on the same `*Trie` must hold its own external lock around all three (`TestTrieConcurrentRemoveAndFind_RequiresExternalLock` documents this by being flagged under `go test -race`).

#### Rules
- Only a path that was previously the target of an `Insert` call (a node with `IsEnd == true`) can be removed; calling `Remove` on a path that was never inserted, on an incomplete/intermediate path, or with malformed input (empty string, no separator) leaves the trie unchanged and returns `false` (`TestTrieRemove_NoMatch_ReturnsFalse`).
- Removing a path prunes every ancestor segment that becomes both childless and not itself an end node as a result, all the way up to (but not including) the root — removing the only path in the trie returns it to `Len() == 0` (`TestTrieRemove_PrunesDeadBranch`).
- An ancestor segment that is still part of another inserted path (shared prefix) is never pruned, even if the path being removed is one of its descendants (`TestTrieRemove_KeepsSharedPrefix`).
- Removing the same path twice returns `true` the first time and `false` the second (`TestTrieRemove_AlreadyRemoved_ReturnsFalse`).

#### Parameters
- 1st parameter: `string` (`removedStr`)

- Description: The previously inserted path to remove, segmented the same way `insertedStr` was at insert time.

- 2nd parameter: `byte` (`sep`)

- Description: The separator byte used to split `removedStr` into segments; must match the separator used when the path was inserted.

#### Returns
- 1st value: `bool`

- Description: `true` if a registered path was found and removed; `false` if no such path existed.

#### Usage

```go
tr := ds.NewTrie()
tr.Insert("/users/:id/", "/users/$/", '/')

ok := tr.Remove("/users/$/", '/')
fmt.Println(ok, tr.Len())
```

Console:
```console
true 0
```

### Find

Walks `path` segment by segment on `sep`, preferring an exact literal match at each level, then — only when `supportParams` is `true` — a `$` (param) child, then a `*` (wildcard) child, falling back to comparing against any sibling segment containing a literal `*` pattern (e.g. `*.html`). While traversing, it also tracks the most specific `*` child passed through so a wildcard fallback is available even when no exact match is found.

#### Rules
- A path must be consumed exactly to a terminal node to count as a match: a path that is an incomplete prefix of a registered route returns `""` for both `matchedRaw` and `wildcardRaw` (`TestTrieFind`, "incomplete path should not match").
- With `supportParams: true`, `$` segments capture their literal path value into `paramVals`, in left-to-right traversal order (`TestTrieFind`, "deep param match"); with `supportParams: false`, `Find` never looks for a `$` child at all, so an inserted `$` segment only matches a query segment that is the literal string `"$"` (`TestTrieFind_ParamSupportDisabled`, `TestTrieFind_ParamSupportEnabled`).
- Once the path passes through a `*` node, that node's `Raw` is reported via `wildcardRaw`, and the match still holds even when the path has extra trailing segments beyond the wildcard route's own length (`TestTrieFind`, "wildcard deep match, extra trailing segments").
- A wildcard match is used as a fallback even when an unrelated, deeper sibling route exists on a different branch that doesn't match the path (`TestTrieFindWildcardFallbackThroughUnrelatedSibling`).

#### Parameters
- 1st parameter: `string` (`path`)

- Description: The path to look up, using the same separator that was used at insert time.

- 2nd parameter: `byte` (`sep`)

- Description: The separator byte used to split `path` into segments.

- 3rd parameter: `bool` (`supportParams`)

- Description: Pass `true` to check for a `$` child at each segment and capture its value; pass `false` to skip that check entirely and treat `$` as an ordinary literal.

#### Returns
- 1st value: `string`

- Description: `Raw` of the node that exactly matches the full path, or `""` if there is no exact match.

- 2nd value: `string`

- Description: `Raw` of the most specific wildcard (`*`) node encountered while traversing the path, or `""` if none was passed through.

- 3rd value: `[]string`

- Description: Values captured for each `$` segment, in the order they were matched. Always `nil` when `supportParams` is `false`.

#### Usage

```go
tr := ds.NewTrie()
tr.Insert("/users/:id/", "/users/$/", '/')

raw, wildcardRaw, params := tr.Find("/users/123/", '/', true)
fmt.Println(raw, wildcardRaw, params)
```

Console:
```console
/users/:id/  [123]
```

### ToJSON

Serializes the trie's shape — every segment's path, `IsEnd`, and children — to a JSON string. Because `Children` is a Go map, the order of sibling entries in the output is not guaranteed to be stable between calls (the keys within each JSON object are always `children`, `isEnd`, `path`, sorted alphabetically by `encoding/json`).

#### Rules
- The root node's JSON object has no `"path"` key, only `"children"`; every other node includes `"path"` (its segment key), `"isEnd"`, and `"children"` (`TestTrieToJSON`).

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
tr.Insert("/users/$/", "/users/$/", '/')
tr.Insert("/feeds/all/", "/feeds/all/", '/')

js, err := tr.ToJSON()
if err != nil {
	panic(err)
}
fmt.Println(js)
```

Console (one possible ordering — sibling order may vary):
```console
{"children":[{"children":[{"children":[],"isEnd":true,"path":"$"}],"isEnd":false,"path":"users"},{"children":[{"children":[],"isEnd":true,"path":"all"}],"isEnd":false,"path":"feeds"}]}
```

## Benchmarks

Captured by running `go test -run=^$ -bench=. -benchmem ./internal/ds/...`. Numbers are machine-dependent and were captured at doc-generation time — re-run the command yourself for a fresh baseline.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/internal/ds
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkMatchWildcard-12         	47284537	        25.25 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_Static-12       	14455305	        81.98 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieFind_WithParam-12    	 6470072	       182.5 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_DeepParam-12    	 3634706	       318.3 ns/op	      64 B/op	       1 allocs/op
BenchmarkTrieFind_NoMatch-12      	 6769974	       182.3 ns/op	       0 B/op	       0 allocs/op
BenchmarkTrieRemove-12            	 1916504	       567.6 ns/op	     144 B/op	       2 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/internal/ds	11.808s
```
