# Building with Ginject — practical reference

Condensed, example-driven knowledge for building features (controllers, providers,
modules, persistence) correctly on the first try, without re-reading framework
source. Pair with `CLAUDE.md` for the route-naming token table.

## 1. The DI shape: Module → Controller → Provider

- **Provider**: any struct with `NewProvider() core.Provider`. Injected into other
  providers/controllers by **struct field type** — reflection-based, no tags.
- **Controller**: any struct with `NewController() core.Controller`. Must embed
  `common.REST` or `common.WS`.
- **Module**: `core.ModuleBuilder().Imports(...).Controllers(...).Providers(...).Build()`.
  - Static module: `var Module = func() *core.Module { ... }`.
  - Dynamic module: a package `Register(opts *Options) *core.Module` factory
    (see `modules/storage`, `modules/cache`, `modules/config`).

**Critical rule — exported fields only**: the framework's reflection-based
injector panics (`can't set value to unexported 'x' field`) if a provider/
controller field that needs injection isn't exported. If a provider needs
shared mutable state, put it behind an **exported pointer field** so every
copy (providers are injected by value) shares the same backing data:

```go
type UserService struct {
    Directory *userDirectory // exported pointer — shared across injected copies
}
```

**Naming collisions**: built-in module services are named generically
(`storage.StoreService`, `cache.CacheService`). If your domain also has a
`StoreService`/similar, alias the import: `dbstorage "github.com/.../modules/storage"`.

## 2. Controllers & routes

Method names are parsed into routes — no annotations. Tokens:
`CREATE/READ/UPDATE/MODIFY/DELETE/PREFLIGHT` → HTTP verb;
`BY` (path param), `OF` (sub-resource, **innermost-first**), `AND` (segment),
`ANY` (wildcard), `VERSION_N` (version). Full table lives in `CLAUDE.md`.

Example: `READ_products_BY_productId_OF_categories_BY_categoryId_OF_store`
→ `GET /store/categories/:categoryId/products/:productId`

Handler params are **resolved by type** — declare what you need and the
framework injects it: `*ctx.Context`, `ctx.Body`, `ctx.Query`, `ctx.Param`,
`ctx.Header`, `ctx.Form`, `ctx.File`, a DTO type (see §3), etc. All re-exported
from root `ginject` package.

```go
func (i StoreController) CREATE_categories_OF_store(c *ctx.Context, dto dto.CategoryDTO) Category
```

## 3. DTOs — validate in the DTO, not the controller

Don't read `ctx.Body.Get("field")` in handlers. Define a DTO implementing
`common.BodyPipeable` (`Transform(ctx.Body, common.ArgumentMetadata) any`);
the framework calls `Transform` and injects the returned value directly as
the handler parameter:

```go
type CategoryDTO struct {
    Name string `bind:"name"`
}

func (d CategoryDTO) Transform(body ctx.Body, arg common.ArgumentMetadata) any {
    bound, _ := body.Bind(d)        // decodes JSON body into the struct via `bind` tags
    dto := bound.(CategoryDTO)
    dto.Name = strings.TrimSpace(dto.Name)
    if dto.Name == "" {
        panic(exception.BadRequestException("name is required"))
    }
    return dto
}
```

- `body.Bind(s)` maps JSON fields to struct fields tagged `bind:"<key>"`.
- Panicking inside `Transform` is safe — it runs inside the handler's
  recover scope and is converted to a proper HTTP error response.
- There is **no built-in `validate:"..."` tag** — write the checks by hand
  and panic with the appropriate `exception.XxxException(message)`.
- Other pipeable kinds exist for query/header/param/form/file/ws-payload
  (`common.QueryPipeable`, etc.) — same `Transform(ctx.Query, common.ArgumentMetadata) any`
  convention, just bind via `query.Bind(d)` instead of `body.Bind(d)`.

**Pagination pattern** — a reusable `PaginationDTO` (`QueryPipeable`) for
list endpoints; clamp instead of reject, since `page=0`/`limit=9999` have
obvious sane interpretations (unlike a malformed body):

```go
type PaginationDTO struct {
    Page  int `bind:"page"`
    Limit int `bind:"limit"`
}
func (d PaginationDTO) Transform(query ctx.Query, arg common.ArgumentMetadata) any {
    bound, _ := query.Bind(d)
    dto := bound.(PaginationDTO)
    if dto.Page < 1 { dto.Page = 1 }
    if dto.Limit < 1 { dto.Limit = 20 } else if dto.Limit > 100 { dto.Limit = 100 }
    return dto
}
```

Pair with a generic `Page[T any] struct { Items []T; Page, Limit, Total, TotalPages int }`
returned from list handlers, e.g. `func (i C) READ_xs(c *ctx.Context, p dto.PaginationDTO) Page[X]`.
For demo-scale stores, fetch all matching docs once via `Find().Where(...).Exec()`,
then slice in Go (`total = len(items)`) — no need for a second count query or
pushing `Skip`/`Limit` into the storage `Query` unless the table is large.

## 4. Exceptions — `Error()` vs `GetResponse()`

`panic(exception.BadRequestException("your message"))` is the standard way
to fail a request from a provider or controller. Two gotchas:

- `ex.Error()` returns the **generic HTTP status text** ("Bad Request",
  "Conflict", …), NOT the message you passed in.
- Your custom message is retrievable only via `ex.GetResponse()`.
- `ex.GetHTTPStatus()` returns `(code int, text string)` derived from
  `ex.GetCode()` (a string like `"400"`).

The framework's default exception filter (`core/default.go`) emits
`{"code", "error": <status text>, "message": <your message>}` — mirror that
shape in any custom interceptor/filter so messages aren't silently dropped.

## 5. Guards & per-route auth

```go
type AuthGuard struct {
    common.Guard
    UserService
}
func (g AuthGuard) NewGuard() AuthGuard { return g }
func (g AuthGuard) CanActivate(c *ctx.Context) bool {
    user, ok := g.UserService.UserBySession(token)
    if !ok { return false }
    c.Request = c.WithContext(context.WithValue(c.Context(), currentUserKey, user))
    return true
}
```

Bind in `NewController`:
- `instance.BindGuard(AuthGuard{})` → applies to **all** handlers.
- `instance.BindGuard(AuthGuard{}, instance.DELETE_sessions)` → specific handlers only.

Use a `const xKey core.WithValueKey = "ns.key"` + `context.WithValue` to pass
data (e.g. current user) from guard → handler; read it back with
`c.Context().Value(xKey)`.

## 6. Interceptors — mind the pre-set status code

The framework sets `c.Code = http.StatusCreated` (201) for **every POST**
*before* the handler runs (`core/http.go`). If a global interceptor's
`aggregation.Error(...)` handler writes the response without calling
`c.Status(...)` first, error responses for POST routes silently come back
as `201` with whatever the handler happened to write. Always do:

```go
aggregation.Error(func(c ginject.Context, e any) any {
    ex, ok := e.(exception.Exception)
    if !ok { c.Status(http.StatusInternalServerError).JSON(...); return nil }
    httpCode, _ := ex.GetHTTPStatus()
    c.Status(httpCode).JSON(ginject.Map{"code": ex.GetCode(), "error": ex.Error(), "message": ex.GetResponse()})
    return nil
})
```

Also note: an `aggregation.Error` handler that returns non-nil suppresses
the framework's own exception filters from running afterward — it fully
owns the error response once it claims the error.

## 7. Built-in modules — prefer these over hand-rolled mocks

### `modules/storage` — file-backed embedded DB
Dynamic module; register with a required `Path`:
```go
var storageModule = storage.Register(&storage.StoreModuleOptions{Path: filepath.Join(cwd, "data", "shop")})
```
Registering also **auto-adds `Path` to the project's `.gitignore`** (walks up
to the nearest `.git`, creates the file if missing, no-ops if the entry is
already present). Set `DisableGitignore: true` to opt out. It's best-effort —
genuine failures print `store: could not add "..." to .gitignore: <err>` to
stderr but never block startup or panic.

Inject `dbstorage.StoreService{ Store *DB-wrapping service }`, then per table:
```go
m := svc.Store.Model("users")
m.Schema(dbstorage.ModelSchema{Fields: []dbstorage.FieldSchema{{Name: "email", Index: true}}}) // call once, e.g. in NewProvider
doc, err := m.Create(map[string]any{"email": email, "name": name})       // -> Document{ID, Data, CreatedAt, UpdatedAt}
doc, err  = m.FindByID(id)
docs, err := m.Find().Where("email", dbstorage.OpEq, email).Exec()       // OpEq/OpNe/OpGt/OpLt/OpContains
err = m.UpdateByID(id, map[string]any{...})  // full replace of Data
err = m.DeleteByID(id)
```
Numbers come back from `Document.Data` as `float64` (JSON-decoded) — cast accordingly.
Write small `xFromDocument(doc) X` helpers to convert to/from domain structs.

### `modules/cache` — TTL key/value store
```go
var cacheModule = cache.Register(&cache.CacheModuleOptions{}) // defaults to in-memory backend
```
Inject `cache.CacheService`; methods take `context.Context` (use
`context.Background()` if you don't have a request-scoped one):
`Get/Set(…, ttl)/SetNX/Delete/Keys/TTL`. Values are `[]byte`. Great fit for
session tokens — set with a TTL and they expire on their own (no manual
cleanup, unlike a plain `map[string]string`).

### `modules/config`
`.env` loader with typed struct binding; register as global, inject
`ConfigService`.

## 8. Wiring it together

Register dynamic modules as imports of your feature module so it stays
self-contained:
```go
var Module = func() *core.Module {
    return core.ModuleBuilder().
        Imports(storageModule, cacheModule).
        Providers(UserService{}, StoreService{}).
        Controllers(UsersController{}, SessionsController{}, StoreController{}).
        Build()
}
```
Then add the feature module to the root app's `Imports(...)` in `main.go`.

## 9. Composing multiple feature modules without import cycles

Imports flatten **upward and recursively**: when module A imports module B,
B's providers and controllers get merged into A's lists (deduped by type —
see `toUniqueControllers`/`genProviderKey` in `core/module.go`, so importing
the same module from several places is safe and creates only one instance).
There's no NestJS-style `Exports`: anything in a module's `Providers(...)` is
available — transitively — to whoever imports it, directly or indirectly.

That gives two ways to share one provider/instance across sibling feature
modules:

1. **Explicit import (preferred — keeps the dependency graph honest)**:
   extract the shared registration into a leaf package that depends on
   nothing feature-specific (e.g. an `infra` package wrapping
   `storage.Register`/`cache.Register` as package-level vars), then have each
   feature module `Imports(infra.StorageModule, infra.CacheModule)`. Since
   `Register` returns one `*core.Module` stored in a package var, every
   importer shares the same underlying `*DB`/cache.
2. **`IsGlobal: true`** on the registered module — its providers land in a
   process-wide pool (`globalProviders`) and are injectable from anywhere
   with no explicit import (see how `modules/config` is typically wired).
   Convenient, but it hides the dependency — prefer explicit imports for
   anything that isn't truly cross-cutting infra.

**Avoiding cycles** when feature A needs something from feature B *and*
B needs something from A (e.g. registering an account also provisions its
store, but every store route requires the account's `AuthGuard`): don't make
A and B import each other. Pick the real direction (`catalog` imports
`accounts` for `AuthGuard`/`CurrentUser`) and pull the *other* direction's
cross-cutting workflow out into a thin **composition-root module** that
imports both and owns nothing but the orchestration:

```go
// shop/module.go — composition root; owns only the controller that needs
// providers from both otherwise-independent feature modules
var Module = func() *core.Module {
    return core.ModuleBuilder().
        Imports(accounts.Module, catalog.Module).
        Controllers(UsersController{}). // embeds accounts.UserService + catalog.StoreService
        Build()
}
```

Resulting graph is a clean DAG: `shop → {accounts, catalog}`,
`catalog → accounts`, both → `infra`. Cross-module injection works exactly
like local injection — embed the provider by its qualified type
(`accounts.UserService`, `catalog.StoreService`); package boundaries don't
matter to the reflection-based injector, only the field's type does.

## 10. Smoke-testing checklist

After wiring a new feature, verify with curl (not just `go build`):
- success codes for each verb (POST→201, GET→200, …)
- validation failures return the **DTO's message**, not a generic one
- domain errors (`409` duplicate, `401` bad credentials, `404` missing,
  `403` unauthenticated) return the right code — easy to get wrong if a
  global interceptor swallows the exception (see §6)
- for persistence-backed features: restart the server and confirm data
  survives (proves it's not an in-memory mock)
