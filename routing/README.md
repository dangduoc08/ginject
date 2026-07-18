# Routing Package

*`routing` builds the method/version/path router that Ginject's core module uses to register, group, and resolve HTTP routes, using the segment trie from `internal/ds`.*

- [Routing Package](#routing-package)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [Constants](#constants)
    - [SERVE](#serve)
    - [ADD, USE, FOR, GROUP](#add-use-for-group)
  - [Package Variables](#package-variables)
    - [OperationsMapHTTPMethods](#operationsmaphttpmethods)
    - [HTTPMethods](#httpmethods)
  - [`RouterItem` Struct](#routeritem-struct)
    - [Method](#method)
    - [Version](#version)
    - [Pattern](#pattern)
    - [Index](#index)
    - [HandlerIndex](#handlerindex)
    - [Handlers](#handlers)
    - [ParamKeys](#paramkeys)
  - [`Router` Struct](#router-struct)
    - [Trie](#trie)
    - [Hash](#hash)
    - [List](#list)
    - [GlobalMiddlewares](#globalmiddlewares)
    - [InjectableHandlers](#injectablehandlers)
  - [Functions](#functions)
    - [NewRouter](#newrouter)
    - [PatternToMethodRouteVersion](#patterntomethodrouteversion)
    - [ToEndpoint](#toendpoint)
    - [MethodRouteVersionToPattern](#methodrouteversiontopattern)
    - [ParseToParamKey](#parsetoparamkey)
  - [`*Router` Methods](#router-methods)
    - [Match](#match)
    - [Group](#group)
    - [Use](#use)
    - [For](#for)
    - [Add](#add)
    - [AddInjectableHandler](#addinjectablehandler)
  - [Benchmarks](#benchmarks)

## Key Features
- Routes are matched through the `internal/ds` trie, so literal, `{param}`, and `*` wildcard segments all resolve through `Match` in one pass
- Independent handler chains per HTTP method and per version tag on the same path
- Three ways to attach handlers — `Use` (global), `For` (route-scoped, any methods), `Add` (the route's main handler) — composed together in call order
- `Group` mounts an entire pre-built sub-router (routes, handlers, and injectable handlers) under a path prefix
- `AddInjectableHandler` registers a handler resolved by reflection instead of the fixed `ctx.Handler` signature, for the DI container

## Usage

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/routing"
)

func main() {
	r := routing.NewRouter()

	getHandler := func(c *ctx.HTTPContext) {}
	r.Add(http.MethodGet, "/users/{id}", "", getHandler)

	isMatched, pattern, paramKeys, paramVals, handlers := r.Match(http.MethodGet, "/users/123/", "")

	fmt.Println("matched:", isMatched)
	fmt.Println("pattern:", pattern)
	fmt.Println("paramKeys:", paramKeys)
	fmt.Println("paramVals:", paramVals)
	fmt.Println("handlers:", len(handlers))
}
```

Console:
```console
matched: true
pattern: /users/{id}/||/[GET]/
paramKeys: map[id:[0]]
paramVals: [123]
handlers: 1
```

## Constants

### SERVE
Type: `string`

Value: `"SERVE"`

A pseudo HTTP method used to register static-file-serving routes. `OperationsMapHTTPMethods[SERVE]` resolves it to `http.MethodGet`, since serving a file is handled as a `GET`.

### ADD, USE, FOR, GROUP
Type: `int`

Values: `ADD = 1`, `USE = 2`, `FOR = 3`, `GROUP = 4` (declared with `iota + 1`)

Internal call-site markers passed to the package's unexported route-registration logic to identify which public method — `Add`, `Use`, `For`, or `Group` — triggered a given registration, so the handler chain is merged differently depending on the caller. No public function accepts one of these as an argument; they're documented here only because they're exported identifiers.

## Package Variables

### OperationsMapHTTPMethods
Type: `map[string]string`

Maps every method this package recognizes to the `net/http` method it should be treated as: each standard method (`GET`, `HEAD`, `POST`, `PUT`, `PATCH`, `DELETE`, `CONNECT`, `OPTIONS`, `TRACE`) maps to itself, and the `SERVE` pseudo-method maps to `http.MethodGet`.

### HTTPMethods
Type: `[]string`

The same set of methods as `OperationsMapHTTPMethods`'s keys (the 9 standard `net/http` methods plus `SERVE`), as an ordered slice — used to register a handler for every method at once, e.g. `for _, m := range routing.HTTPMethods { r.Add(m, route, version, handler) }`.

## `RouterItem` Struct

Holds everything `Match` needs once the trie has resolved a path: which method/version it was registered for, and its handler chain. Several `RouterItem`s share the same `Index` (and the same trie leaf) when the same path is registered under different methods or versions.

### Method
Type: `string`

Default: the `method` argument passed to `Add` (or to `For`/`Use` for an already-registered route)

Required: `true`

The HTTP method (or `SERVE`) this item is registered for.

### Version
Type: `string`

Default: the `version` argument passed in; `""` if none

Required: `false`

The version tag this item is registered for. `Match` only returns an item whose `Version` exactly matches the version it was called with.

### Pattern
Type: `string`

Default: `MethodRouteVersionToPattern(method, route, version)`

Required: `true`

The canonical pattern string identifying this exact method+route+version combination; this is what `Match` returns as its 2nd value on a successful match.

### Index
Type: `int`

Default: the position of this item's route within the router's `List`

Required: `true`

The id this item's route resolves to in the embedded trie; shared by every `RouterItem` registered for the same route path, whatever their method/version.

### HandlerIndex
Type: `int`

Default: `-1`

Required: `false`

Index, within `Handlers`, of the slot last written by `Add`. `-1` means `Add` hasn't registered a main handler for this method/route/version yet.

### Handlers
Type: `[]ctx.Handler`

Default: `nil` until the first `Use`, `For`, or `Add` call touches this method/route/version

Required: `false`

The full handler chain — global middlewares, route-scoped middlewares, and the main handler, in the order they were attached — that `Match` returns as its 5th value.

### ParamKeys
Type: `map[string][]int`

Default: `nil` unless the route contains `{name}` placeholders

Required: `false`

For each named parameter in the route, the positions (in `$`-segment traversal order) where it appears; built by `ParseToParamKey` when the item is registered.

## `Router` Struct

The router itself: an embedded trie plus the bookkeeping `Add`, `Use`, `For`, `Group`, and `Match` need.

### Trie
Type: `*ds.Trie`

Default: `ds.NewTrie()`

Required: `false`

Embedded anonymously, so `ds.Trie`'s exported methods (`Len`, `Insert`, `Find`, `ToJSON`) are promoted directly onto `Router` (e.g. `router.Len()`). `Add` inserts every registered route into this trie, and `Match` resolves incoming paths through it; see `internal/ds`'s own README for those methods' documented behavior.

### Hash
Type: `map[string][]RouterItem`

Default: `make(map[string][]RouterItem)`

Required: `false`

Every `RouterItem` registered so far, keyed by the route's normalized endpoint path (`ToEndpoint(route)`) — one key can hold several items, one per method/version combination registered for that path.

### List
Type: `[]string`

Default: `nil`

Required: `false`

Every distinct normalized route path registered so far, in registration order; a path is appended the first time it's seen, and its index in this slice becomes that route's `Index`/trie id.

### GlobalMiddlewares
Type: `[]ctx.Handler`

Default: `[]ctx.Handler{}`

Required: `false`

Handlers accumulated via `Use`. They're prepended ahead of a route's own handler when that route is registered with no handler yet, and are retroactively appended to every already-registered route's `Handlers` chain.

### InjectableHandlers
Type: `map[string]any`

Default: `make(map[string]any)`

Required: `false`

Handlers registered via `AddInjectableHandler`, keyed by `MethodRouteVersionToPattern(method, route, version)`. `Group` also copies a sub-router's entries onto the receiver, re-keyed under the group prefix.

## Functions

### NewRouter

Creates an empty `*Router` ready for `Add`, `Use`, `For`, `Group`, `Match`, and `AddInjectableHandler`.

#### Parameters
None.

#### Returns
- 1st value: `*Router`

- Description: A router with an empty trie, an empty `Hash`, a `nil` `List`, an empty `GlobalMiddlewares` slice, and an empty `InjectableHandlers` map.

#### Usage

```go
r := routing.NewRouter()
```

### PatternToMethodRouteVersion

Parses a pattern string produced by `MethodRouteVersionToPattern` back into its method, route, and version components — the inverse of `MethodRouteVersionToPattern`.

#### Rules
- Recovers the exact `(method, route, version)` triple that produced the pattern, e.g. `"/users/$/|v2|/[POST]/"` → `("POST", "/users/$", "v2")` (`TestPatternToMethodRouteVersion`).
- An empty version segment (`||`) parses back to an empty `version` string, e.g. `"/users/$/||/[GET]/"` → version `""` (`TestPatternToMethodRouteVersion`).

#### Parameters
- 1st parameter: `string` (`pattern`)

- Description: A pattern string in the `<route>/<|version|>/<[METHOD]>/` shape produced by `MethodRouteVersionToPattern`.

#### Returns
- 1st value: `string`

- Description: The HTTP method extracted from the pattern's `[METHOD]` segment.

- 2nd value: `string`

- Description: The route path extracted from the pattern, with the version/method suffix removed.

- 3rd value: `string`

- Description: The version tag extracted from the pattern's `|version|` segment, or `""` if it was empty.

#### Usage

```go
method, route, version := routing.PatternToMethodRouteVersion("/users/$/|v2|/[POST]/")
fmt.Println(method, route, version)
```

Console:
```console
POST /users/$ v2
```

### ToEndpoint

Normalizes a raw path string into a router-ready endpoint: always wrapped in leading/trailing `/`, with whitespace stripped and repeated `/`/`*` collapsed.

#### Rules
- Always wraps the result in a leading and trailing `/` (`"users"` → `"/users/"`) (`TestToEndpoint`).
- Collapses consecutive `/` characters and consecutive `*` characters down to one each (`"//users//"` → `"/users/"`; `"/a/**/b/"` → `"/a/*/b/"`) (`TestToEndpoint`).
- Strips ASCII whitespace anywhere in the input, including leading/trailing (`" /users/ "` → `"/users/"`) (`TestToEndpoint`).

#### Parameters
- 1st parameter: `string` (`str`)

- Description: The raw path to normalize.

#### Returns
- 1st value: `string`

- Description: The normalized endpoint path.

#### Usage

```go
fmt.Println(routing.ToEndpoint("//users//"))
fmt.Println(routing.ToEndpoint("/a/**/b/"))
```

Console:
```console
/users/
/a/*/b/
```

### MethodRouteVersionToPattern

Builds the canonical pattern string `Router` stores on each `RouterItem` and uses to look up registered routes by method/route/version.

#### Rules
- Produces a `"<endpoint>/<|version|>/<[METHOD]>/"`-shaped string, e.g. `(GET, "/users/{userId}", "")` → `"/users/{userId}/||/[GET]/"`; with a version: `(POST, "/users/{userId}", "v2")` → `"/users/{userId}/|v2|/[POST]/"` (`TestMethodRouteVersionToPattern`).
- An empty `method` still produces a pattern with empty brackets `[]` rather than omitting the method segment, e.g. `("", "/feeds/all", "")` → `"/feeds/all/||/[]/"` (`TestMethodRouteVersionToPattern`).

#### Parameters
- 1st parameter: `string` (`method`)

- Description: The HTTP method to embed in the pattern.

- 2nd parameter: `string` (`route`)

- Description: The route path to normalize (via `ToEndpoint`) and embed.

- 3rd parameter: `string` (`version`)

- Description: The version tag to embed; pass `""` for no version.

#### Returns
- 1st value: `string`

- Description: The combined pattern string.

#### Usage

```go
fmt.Println(routing.MethodRouteVersionToPattern(http.MethodPost, "/users/{userId}", "v2"))
```

Console:
```console
/users/{userId}/|v2|/[POST]/
```

### ParseToParamKey

Replaces every `{name}` placeholder in a route with a literal `$` segment (the form the trie matches on) and records where each name appears.

#### Rules
- Replaces every `{paramName}` placeholder with `$`, and returns a `map[string][]int` recording, for each name, the zero-based index (in order of appearance) of every placeholder using that name, e.g. `"/users/{userId}/friends/{friendId}/"` → `"/users/$/friends/$/"` with `keys["userId"] == [0]` and `keys["friendId"] == [1]` (`TestParseToParamKey`).
- A string with no `{...}` placeholders is returned unchanged, with an empty param-key map (`"/plain/route/"` → `("/plain/route/", map[string][]int{})`) (`TestParseToParamKey`).

#### Parameters
- 1st parameter: `string` (`str`)

- Description: The route path to scan for `{name}` placeholders.

#### Returns
- 1st value: `string`

- Description: The route with every `{name}` placeholder replaced by `$`.

- 2nd value: `map[string][]int`

- Description: For each parameter name, the list of `$`-segment positions it occupies.

#### Usage

```go
str, keys := routing.ParseToParamKey("/users/{userId}/friends/{friendId}/")
fmt.Println(str, keys)
```

Console:
```console
/users/$/friends/$/ map[friendId:[1] userId:[0]]
```

## `*Router` Methods

### Match

Resolves an incoming `(method, route, version)` request against every route registered on the router and returns its handler chain.

#### Rules
- Returns `isMatched = false` when there's no `RouterItem` for the resolved route under the exact `(method, version)` pair — even if the same path is registered under a different method or version (`TestRouterMatchSamePathDifferentMethodAndVersion`: a `DELETE` that was never registered, and a `POST` + `"v2"` combination that was never registered, both fail to match).
- The 4th return value (`paramVals`) holds the literal values captured for each `{name}` segment, in the order the segments appear in the route (`TestRouterMatchSamePathDifferentMethodAndVersion`: matching `/users/123/` against `/users/{id}` yields `paramVals[0] == "123"`).
- The 2nd return value equals `MethodRouteVersionToPattern(method, <registered route>, version)` for whichever registered route — literal, `{param}`, or `*` wildcard — the path resolves to (`TestRouterMatch`, covering static, single- and multi-param, and wildcard/literal-pattern routes such as `in*.html`).

#### Parameters
- 1st parameter: `string` (`method`)

- Description: The HTTP method of the incoming request.

- 2nd parameter: `string` (`route`)

- Description: The incoming request path to resolve.

- 3rd parameter: `string` (`version`)

- Description: The version tag to match against; `""` for no version.

#### Returns
- 1st value: `bool`

- Description: Whether a registered handler was found for this exact method/version.

- 2nd value: `string`

- Description: The matched route's canonical `Pattern`, or `""` if there's no match.

- 3rd value: `map[string][]int`

- Description: The matched route's `ParamKeys`, or `nil` if there's no match.

- 4th value: `[]string`

- Description: The captured parameter values, in route order.

- 5th value: `[]ctx.Handler`

- Description: The matched route's handler chain, or `nil` if there's no match.

#### Usage

```go
isMatched, pattern, paramKeys, paramVals, handlers := r.Match(http.MethodGet, "/users/123/", "")
fmt.Println(isMatched, pattern, paramKeys, paramVals, len(handlers))
```

Console:
```console
true /users/{id}/||/[GET]/ map[id:[0]] [123] 1
```

### Group

Mounts every route (and injectable handler) from one or more already-built sub-routers onto the receiver, under a path prefix.

#### Rules
- For every route registered on each `subRouter`, re-registers it on the receiver under `prefix + route`, for the same method and version, preserving that route's handler chain (`TestRouterGroup`: grouping a sub-router with `/users/update/{userId}` under prefix `/v1` makes `PATCH /v1/users/update/123/` resolve to the pattern for `/v1/users/update/{userId}`).
- Multiple sub-routers can be grouped under the same prefix in one call; routes from each are merged onto the receiver (`TestRouterGroup` groups two sub-routers together under `/v1` in a single `Group` call).

#### Parameters
- 1st parameter: `string` (`prefix`)

- Description: The path prefix to mount every sub-router's routes under.

- 2nd parameter: `...*Router` (`subRouters`)

- Description: One or more already-built routers whose routes (and injectable handlers) should be merged onto the receiver.

#### Returns
- 1st value: `*Router`

- Description: The receiver, so calls can be chained.

#### Usage

```go
v1 := routing.NewRouter()
v1.Add(http.MethodPatch, "/users/update/{userId}", "", func(c *ctx.HTTPContext) {})

gr := routing.NewRouter()
gr.Group("/v1", v1)

_, pattern, _, _, _ := gr.Match(http.MethodPatch, "/v1/users/update/123/", "")
fmt.Println(pattern)
```

Console:
```console
/v1/users/update/{userId}/||/[PATCH]/
```

### Use

Registers handlers as global middlewares for every route on this router.

#### Rules
- Appends the given handlers to `GlobalMiddlewares`, and retroactively appends them to the `Handlers` chain of every route already registered on the router (`TestRouterMiddleware`: a second `Use(handler1)` call made after a route already has 4 handlers grows that route's chain to 5).
- Routes added *after* `Use` has accumulated global middlewares get those middlewares prepended ahead of their own handler, as long as no handler had been registered yet for that exact method/route/version (`TestRouterMiddleware`, router 0).
- Returns the receiver, so calls can be chained, e.g. `gr.Use(handler4).Use(handler2).Use(handler1)` (`TestRouterMiddleware`).

#### Parameters
- 1st parameter: `...ctx.Handler` (`handlers`)

- Description: Handlers to register as global middlewares.

#### Returns
- 1st value: `*Router`

- Description: The receiver, so calls can be chained.

#### Usage

```go
r.Use(func(c *ctx.HTTPContext) {
	c.Next()
})
```

### For

Returns a function that registers handlers for a route across a specific set of HTTP methods.

#### Rules
- The returned closure appends its handlers to the `Handlers` chain of every method in `methodInclusions`, for the given `route`/`version` (`TestRouterMiddleware`: `r0.For(HTTPMethods, "/test0", "")(handler1)` adds `handler1` to every HTTP method's chain for `/test0`).
- If `GlobalMiddlewares` were already set via `Use` before any handler existed for that route, the appended handlers land after them in the chain (`TestRouterMiddleware`, router 1).

#### Parameters
- 1st parameter: `[]string` (`methodInclusions`)

- Description: The HTTP methods to register the handlers under.

- 2nd parameter: `string` (`route`)

- Description: The route path.

- 3rd parameter: `string` (`version`)

- Description: The version tag; `""` for no version.

#### Returns
- 1st value: `func(handlers ...ctx.Handler) *Router`

- Description: A function that, when called with handlers, registers them and returns the receiver for chaining.

#### Usage

```go
r.For([]string{http.MethodGet, http.MethodPost}, "/users/{id}", "")(func(c *ctx.HTTPContext) {
	c.Next()
})
```

### Add

Registers the main handler for a single `(method, route, version)` combination, inserting the route into the embedded trie.

#### Rules
- Each call to `Add` inserts the route into the router's embedded trie, so the trie's `Len()` (promoted from `ds.Trie`) reflects the total number of distinct path segments across every route added so far, with shared prefixes counted once (`TestRouteAdd`: 4 routes sharing prefixes produce `Len() == 11`).
- The handler passed to `Add` is appended after any handlers already contributed by `Use`/`For` for that exact method/route/version (`TestRouterMiddleware`, routers 0 and 1).

#### Parameters
- 1st parameter: `string` (`method`)

- Description: The HTTP method to register the handler under.

- 2nd parameter: `string` (`route`)

- Description: The route path, e.g. `/users/{id}`.

- 3rd parameter: `string` (`version`)

- Description: The version tag; `""` for no version.

- 4th parameter: `ctx.Handler` (`handler`)

- Description: The main handler for this route.

#### Returns
- 1st value: `*Router`

- Description: The receiver, so calls can be chained.

#### Usage

```go
r.Add(http.MethodGet, "/users/{id}", "", func(c *ctx.HTTPContext) {})
```

### AddInjectableHandler

Registers a handler resolved by reflection (rather than the fixed `ctx.Handler` signature) for a route, for use by the DI container.

#### Rules
- Stores `handler` in `InjectableHandlers`, keyed by `MethodRouteVersionToPattern(method, route, version)`, and also calls `Add` so the route becomes matchable via `Match` (`TestAddInjectableHandler`).
- Panics if `handler` is `nil` (`TestAddInjectableHandlerPanicsOnNil`).
- Panics if `handler` is not a function, even if non-nil (`TestAddInjectableHandlerPanicsOnNonFunc`).

#### Parameters
- 1st parameter: `string` (`method`)

- Description: The HTTP method to register the handler under.

- 2nd parameter: `string` (`route`)

- Description: The route path.

- 3rd parameter: `string` (`version`)

- Description: The version tag; `""` for no version.

- 4th parameter: `any` (`handler`)

- Description: The handler to store; must be a non-nil function.

#### Returns
- 1st value: `*Router`

- Description: The receiver, so calls can be chained.

#### Usage

```go
r.AddInjectableHandler(http.MethodGet, "/users/{userId}", "", func(userID string) {})
```

## Benchmarks

Captured by running `go test -run=^$ -bench=. -benchmem ./routing/...`. Numbers are machine-dependent and were captured at doc-generation time — re-run the command yourself for a fresh baseline.

```console
goos: darwin
goarch: amd64
pkg: github.com/dangduoc08/ginject/routing
cpu: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
BenchmarkPatternToMethodRouteVersion-12    	 7198441	       159.5 ns/op	       0 B/op	       0 allocs/op
BenchmarkParseToParamKey-12                	197307458	         6.123 ns/op	       0 B/op	       0 allocs/op
BenchmarkToEndpoint-12                     	12655910	        88.90 ns/op	      32 B/op	       1 allocs/op
BenchmarkMethodRouteVersionToPattern-12    	 6494689	       185.7 ns/op	      60 B/op	       3 allocs/op
BenchmarkRouterAdd-12                      	  375076	      4246 ns/op	    2234 B/op	      23 allocs/op
BenchmarkRouterMatch_Static-12             	 5007302	       207.5 ns/op	      16 B/op	       1 allocs/op
BenchmarkRouterMatch_Param-12              	 2435484	       461.2 ns/op	     112 B/op	       2 allocs/op
BenchmarkRouterMatch_NoMatch-12            	   57026	     20988 ns/op	      24 B/op	       1 allocs/op
BenchmarkRouterUse-12                      	   10000	    119744 ns/op	   79594 B/op	     801 allocs/op
PASS
ok  	github.com/dangduoc08/ginject/routing	18.081s
```
