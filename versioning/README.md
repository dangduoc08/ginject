# Versioning

*Versioning is a part of `Ginject` framework, it extracts an API version string from an incoming request using one of four pluggable strategies: query parameter, header, media type, or a custom extractor function.*

- [Versioning](#versioning)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [Constants](#constants)
  - [`Versioning` Fields](#versioning-fields)
    - [Type](#type)
    - [Key](#key)
    - [DefaultVersion](#defaultversion)
    - [Extractor](#extractor)
  - [`*Versioning` Methods](#versioning-methods)
    - [GetTypeString](#gettypestring)
      - [Returns](#returns)
      - [Usage](#usage-1)
    - [GetVersion](#getversion)
      - [Parameters](#parameters)
      - [Returns](#returns-1)
      - [Usage](#usage-2)
      - [Rules](#rules)
  - [Benchmarks](#benchmarks)

## Key Features
- Four extraction strategies: query parameter, header, media type, or a custom function
- Pluggable `Extractor` for strategies the built-ins don't cover
- `NeutralVersion` sentinel for routes that should match regardless of requested version
- Falls back to `DefaultVersion` whenever the requested strategy finds nothing

## Usage

`Versioning` is wired into the framework via `core.App.EnableVersioning`. Construct it and chain it off your app:

```go
package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/versioning"
)

func main() {
	app := core.New()

	app.
		EnableVersioning(versioning.Versioning{
			Type: versioning.HeaderVersion,
			Key:  "X-Api-Version",
		}).
		EnableDevtool()

	app.Create(
		core.ModuleBuilder().Build(),
	)
}
```

With the above config, a controller method versioned with the `VERSION_1` route token (see the root `CLAUDE.md` for the naming convention) only matches requests sending `X-Api-Version: 1`. Requests with no matching version fall back to routes registered under `versioning.NeutralVersion`.

`Versioning` can also be used standalone, independent of `core`, by calling `GetVersion` directly with a `*ctx.Context`:

```go
v := &versioning.Versioning{
	Type:           versioning.QueryVersion,
	Key:            "version",
	DefaultVersion: "v1",
}

requestedVersion := v.GetVersion(c) // c is *ctx.Context
```

## Constants

```go
const (
	QueryVersion  = iota + 1 // extract from a query parameter
	HeaderVersion            // extract from a header
	CustomVersion            // extract via a caller-supplied Extractor
	MediaType                // extract from an Accept header parameter
)

const NeutralVersion = "NEUTRAL"
```

`QueryVersion`, `HeaderVersion`, `CustomVersion`, and `MediaType` are the valid values for the [`Type`](#type) field. `NeutralVersion` is a sentinel string meant to be used as a route's version (or as `DefaultVersion`) to mark it as matching regardless of the version requested by the caller.

## `Versioning` Fields

### Type
Type: `int`

Default: `0` (matches none of the defined constants)

Required: `true`

Selects which extraction strategy `GetVersion` uses. Must be one of [`QueryVersion`](#constants), [`HeaderVersion`](#constants), [`CustomVersion`](#constants), or [`MediaType`](#constants). Any other value (including the zero value) makes `GetVersion` always return [`DefaultVersion`](#defaultversion) and makes [`GetTypeString`](#gettypestring) return `""`.

```go
versioning.Versioning{
	Type: versioning.QueryVersion,
}
```

### Key
Type: `string`

Default: `""`

Required: `false`

The parameter or header name to look up. Required (in practice) for `QueryVersion` (query parameter name), `HeaderVersion` (header name), and `MediaType` (the parameter name inside the `Accept` header, e.g. `v` in `Accept: application/json;v=2`). Unused for `CustomVersion`.

```go
versioning.Versioning{
	Type: versioning.MediaType,
	Key:  "v", // matches "application/json;v=2"
}
```

### DefaultVersion
Type: `string`

Default: `""`

Required: `false`

Returned by [`GetVersion`](#getversion) whenever the selected strategy finds nothing — the key is absent from the query/header, the `Accept` header doesn't carry the parameter, or `Extractor` is `nil`. Can be set to [`NeutralVersion`](#constants) so unversioned requests match version-agnostic routes.

```go
versioning.Versioning{
	Type:           versioning.HeaderVersion,
	Key:            "X-Api-Version",
	DefaultVersion: versioning.NeutralVersion,
}
```

### Extractor
Type: `ExtractorHandler` (`func(*ctx.Context) string`)

Default: `nil`

Required: `false`

Only consulted when [`Type`](#type) is `CustomVersion`. Called with the request `*ctx.Context`; its return value is used as-is, even if it's an empty string. If `Extractor` is `nil`, `GetVersion` returns `DefaultVersion` instead of calling it.

```go
versioning.Versioning{
	Type: versioning.CustomVersion,
	Extractor: func(c *ctx.Context) string {
		return c.Param().Get("tenant")
	},
}
```

## `*Versioning` Methods

### GetTypeString

Returns a human-readable name for the receiver's [`Type`](#type).

#### Returns
- 1st value: `string`

- Description: `"query"`, `"header"`, `"custom"`, or `"media_type"` for the four defined constants; `""` for any other value.

#### Usage

```go
v := &versioning.Versioning{Type: versioning.HeaderVersion}
v.GetTypeString() // "header"
```

### GetVersion

Extracts the requested API version from a request context, using the strategy selected by [`Type`](#type).

#### Parameters
- 1st parameter: `*ctx.Context`

- Description: The request context to extract the version from.

#### Returns
- 1st value: `string`

- Description: The extracted version, or [`DefaultVersion`](#defaultversion) if the strategy found nothing.

#### Usage

```go
v := &versioning.Versioning{
	Type:           versioning.QueryVersion,
	Key:            "version",
	DefaultVersion: "v1",
}

version := v.GetVersion(c) // c is *ctx.Context
```

#### Rules
- `QueryVersion`: returns the value of query parameter `Key` if present and non-empty; otherwise returns `DefaultVersion`.
- `HeaderVersion`: returns the value of header `Key` if present and non-empty; otherwise returns `DefaultVersion`.
- `CustomVersion`: if `Extractor` is non-`nil`, returns `Extractor(c)` directly — even if that result is `""`. If `Extractor` is `nil`, returns `DefaultVersion`.
- `MediaType`: parses the `Accept` header for a `Key=value` parameter (value ends at the next `;`, surrounding whitespace trimmed); returns it if non-empty, otherwise returns `DefaultVersion`. Also returns `DefaultVersion` when the `Accept` header is absent entirely.
- `DefaultVersion` may be set to [`NeutralVersion`](#constants); `GetVersion` returns it like any other string.

## Benchmarks

Captured at doc-generation time on an Intel Core i7-9750H (results are machine-dependent — re-run `go test -bench=. -benchmem ./versioning/...` on your own hardware for representative numbers):

```
BenchmarkGetVersion_Query-12        	73886743	        14.62 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetVersion_Header-12       	27834199	        45.27 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetVersion_Custom-12       	374553765	         3.310 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetVersion_MediaType-12    	16623618	        60.69 ns/op	       0 B/op	       0 allocs/op
```

All four strategies are allocation-free.
