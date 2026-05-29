# storage

Embedded, crash-safe document database for Ginject applications. Runs inside your process — no external database required. Data is stored as append-only binary segment files under a local directory.

---

## Table of Contents

- [Quick start](#quick-start)
- [Ginject module integration](#ginject-module-integration)
- [Opening a database directly](#opening-a-database-directly)
- [Defining a schema](#defining-a-schema)
- [CRUD](#crud)
- [Querying](#querying)
- [Text search](#text-search)
- [Transactions](#transactions)
- [Hooks](#hooks)
- [Watching for changes](#watching-for-changes)
- [Maintenance — Flush and Compact](#maintenance--flush-and-compact)
- [Use cases](#use-cases)
- [Concurrency model](#concurrency-model)
- [Race condition scenarios and how they are handled](#race-condition-scenarios-and-how-they-are-handled)
- [Atomic operations](#atomic-operations)
- [Crash recovery](#crash-recovery)
- [Storage layout](#storage-layout)
- [Errors](#errors)

---

## Quick start

```go
db, err := storage.Open("./data")
if err != nil {
    log.Fatal(err)
}
defer db.Close()

users := db.Model("users")

// create
doc, _ := users.Create(map[string]any{
    "name":  "Alice",
    "email": "alice@example.com",
    "role":  "admin",
})
fmt.Println(doc.ID) // e.g. "a3f1c2b4..."

// read
found, _ := users.FindByID(doc.ID)

// update
_ = users.UpdateByID(doc.ID, map[string]any{"role": "superadmin"})

// delete
_ = users.DeleteByID(doc.ID)
```

---

## Ginject module integration

Register the module once — typically as global so every module can inject `StoreService`:

```go
// app module
var AppModule = func() *core.Module {
    return core.ModuleBuilder().
        Imports(
            storage.Register(&storage.StoreModuleOptions{
                IsGlobal: true,
                Path:     "./data",
            }),
        ).
        Controllers(UserController{}).
        Build()
}
```

Inject `StoreService` by embedding it in a provider or controller:

```go
type UserService struct {
    storage.StoreService
}

func (s *UserService) CreateUser(name, email string) (storage.Document, error) {
    return s.Model("users").Create(map[string]any{
        "name":  name,
        "email": email,
    })
}
```

`StoreService` exposes `Model`, `Tx`, `Flush`, and `Compact` — the same methods as `*DB`.

---

## Opening a database directly

Use `Open` when you need a database outside of the DI container (tests, scripts, CLI tools):

```go
db, err := storage.Open("/var/lib/myapp/db")
if err != nil {
    // directory is created automatically; error only on permission problems
    log.Fatal(err)
}
defer db.Close()
```

`Open` scans existing segment files and rebuilds the in-memory primary index. It is safe to call on an existing directory — data is preserved.

---

## Defining a schema

A schema is optional. Without one, all queries use a full primary-index scan and text search returns nothing. With a schema, equality queries use a secondary index and full-text search uses an inverted index.

**Call `Schema` before the application starts serving requests.** It rebuilds the secondary and text indexes from existing data on every call.

```go
db.Model("posts").Schema(storage.ModelSchema{
    Fields: []storage.FieldSchema{
        {Name: "status",  Index: true},          // secondary index
        {Name: "authorID", Index: true},          // secondary index
        {Name: "title",   Search: true},          // text search
        {Name: "body",    Search: true},          // text search
        {Name: "tags",    Index: true, Search: true}, // both
    },
})
```

`Index: true` — builds a field-value → ID mapping. Used by `Where("field", OpEq, value)`.  
`Search: true` — builds a term → ID inverted index. Used by `Search("query")`.

Multiple fields can be indexed simultaneously. There is no limit.

---

## CRUD

```go
m := db.Model("users")

// Create — generates a 32-char hex ID; sets CreatedAt and UpdatedAt
doc, err := m.Create(map[string]any{
    "name":  "Bob",
    "email": "bob@example.com",
    "score": 42,
})

// FindByID
doc, err := m.FindByID(doc.ID)
// ErrNotFound if the ID does not exist

// UpdateByID — replaces the entire Data map; preserves CreatedAt
err := m.UpdateByID(doc.ID, map[string]any{
    "name":  "Bob",
    "email": "bob@example.com",
    "score": 100,
})
// ErrNotFound if the ID does not exist

// DeleteByID
err := m.DeleteByID(doc.ID)
// ErrNotFound if the ID does not exist
```

`Document` fields:

```go
type Document struct {
    ID        string         // immutable after creation
    Data      map[string]any // user-provided fields
    CreatedAt time.Time      // set on Create; never modified
    UpdatedAt time.Time      // updated on every UpdateByID
}
```

---

## Querying

```go
m := db.Model("users")

// all documents
docs, err := m.Find().Exec()

// filter + pagination
docs, err := m.Find().
    Where("role", storage.OpEq, "admin").
    Where("active", storage.OpEq, "true").
    Limit(20).
    Skip(40). // page 3 of 20
    Exec()
```

### Supported operators

| Constant | Meaning |
|----------|---------|
| `OpEq` | field == value |
| `OpNe` | field != value |
| `OpGt` | field > value (lexicographic) |
| `OpLt` | field < value (lexicographic) |
| `OpContains` | field contains value as a substring |

Multiple `Where` calls are combined with AND.

### Index usage

If the first `Where` clause uses `OpEq` on a field declared with `Index: true`, the secondary index is used to narrow the candidate set before the remaining conditions are evaluated. Otherwise a full scan of the primary index runs.

```go
// Uses secondary index (role is indexed)
m.Find().Where("role", storage.OpEq, "admin").Where("active", storage.OpEq, "true").Exec()

// Full scan (role is not indexed)
m.Find().Where("role", storage.OpEq, "admin").Exec()
```

---

## Text search

```go
// Requires Search: true on at least one field in the schema
docs, err := m.Search("embedded golang database")
```

- Tokens are lowercased and split on non-alphanumeric characters.
- Tokens shorter than 2 characters are ignored.
- All terms must be present in a document for it to be returned (**AND semantics**).
- Search returns all matching documents (no limit/skip); add your own slice if you need pagination.

```go
// Search then paginate manually
results, _ := m.Search("golang")
page := results[0:min(10, len(results))]
```

---

## Transactions

A transaction groups multiple writes across one or more tables into an all-or-nothing operation. Operations are buffered in memory and written atomically to disk when the callback returns `nil`.

```go
err := db.Tx(func(tx *storage.Tx) error {
    // debit
    if err := tx.Model("accounts").UpdateByID(fromID, map[string]any{
        "balance": newFromBalance,
    }); err != nil {
        return err // nothing is written
    }
    // credit
    if err := tx.Model("accounts").UpdateByID(toID, map[string]any{
        "balance": newToBalance,
    }); err != nil {
        return err
    }
    // audit log
    _, err := tx.Model("transfers").Create(map[string]any{
        "from": fromID, "to": toID, "amount": amount,
    })
    return err
})
```

If the callback returns any error, `Tx` returns that error and no data is written.

`TxModel` supports `Create`, `UpdateByID`, and `DeleteByID`.

### What "atomic" means here

When the callback returns `nil`:

1. A `TX_BEGIN` marker is written to the segment.
2. All buffered records are written in order.
3. A `TX_COMMIT` marker is written and `fsync` is called.
4. In-memory indexes are updated.

Steps 1–3 happen under the engine write lock. If the process crashes between step 3 and step 4, the segment already has the full committed transaction. On the next `Open`, the recovery scan finds the `TX_COMMIT` and replays the indexes — no data is lost.

---

## Hooks

Hooks let you intercept CRUD events to add auditing, validation, or field injection.

```go
// inject a "createdBy" field before every create
db.Pre("create", func(hc *storage.HookCtx) {
    hc.Data["createdBy"] = "system"
    hc.Data["createdAt"] = time.Now().Format(time.RFC3339)
})

// log after every delete
db.Post("delete", func(hc *storage.HookCtx) {
    log.Printf("deleted %s/%s", hc.Table, hc.ID)
})
```

Valid event names: `"create"`, `"update"`, `"delete"`, `"find"`.

`HookCtx` fields:

```go
type HookCtx struct {
    Event string         // "create" | "update" | "delete" | "find"
    Table string         // table name
    ID    string         // document ID (empty for pre-create)
    Data  map[string]any // mutable on pre-hooks; read-only intent on post-hooks
}
```

Pre-create hooks receive an empty `ID` because the ID is generated after the pre-hook runs.

### Global middleware

```go
db.Use(func(hc *storage.HookCtx, next func()) {
    start := time.Now()
    next()
    log.Printf("%s %s took %v", hc.Event, hc.Table, time.Since(start))
})
```

`Use` wraps every pre-hook invocation. `next()` must be called to continue the chain.

---

## Watching for changes

`Watch` delivers real-time events after each successful write on a model:

```go
m := db.Model("inventory")
unsubscribe := m.Watch(func(e storage.Event) {
    switch e.Type {
    case storage.EventCreate:
        cache.Invalidate(e.Doc.ID)
    case storage.EventUpdate:
        notify(e.Doc)
    case storage.EventDelete:
        removeFromSearch(e.Doc.ID)
    }
})
defer unsubscribe() // remove watcher when no longer needed
```

- Events fire synchronously after the write lock is released, in the goroutine that performed the write.
- Multiple watchers on the same model are all notified.
- Calling the returned unsubscribe function is safe from any goroutine.
- Slow watcher callbacks block the caller; run heavy work in a goroutine inside the callback.

```go
m.Watch(func(e storage.Event) {
    go func(ev storage.Event) {
        sendWebhook(ev) // off the hot path
    }(e)
})
```

---

## Maintenance — Flush and Compact

### Flush

Syncs the current segment file of every open table to the OS page cache and then to disk.

```go
_ = db.Flush()
```

Call periodically in long-running applications if you do not use transactions (which `fsync` on every commit). For transactional workloads `fsync` already runs per commit and `Flush` is redundant.

### Compact

Rewrites each table's segment files, keeping only the latest version of each document and discarding deleted records. Reduces disk usage and speeds up the next startup (fewer records to replay).

```go
_ = db.Compact()
```

**Compact briefly holds the write lock for each table.** Reads and writes to that table are blocked for the duration. On tables with millions of records this may take seconds. Schedule it during off-peak hours.

```go
// example: compact at 03:00 every day
go func() {
    for {
        now := time.Now()
        next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, now.Location())
        time.Sleep(time.Until(next))
        if err := db.Compact(); err != nil {
            log.Println("compact error:", err)
        }
    }
}()
```

---

## Use cases

### 1. User and session management for a small web app

```go
users := db.Model("users").Schema(storage.ModelSchema{
    Fields: []storage.FieldSchema{
        {Name: "email", Index: true},
        {Name: "role",  Index: true},
    },
})

// register
doc, _ := users.Create(map[string]any{
    "email":    "alice@example.com",
    "role":     "member",
    "password": hashPassword("secret"),
})

// login — find by email (uses secondary index, O(1) lookup)
results, _ := users.Find().Where("email", storage.OpEq, "alice@example.com").Limit(1).Exec()
if len(results) == 0 {
    // not found
}

// promote to admin
_ = users.UpdateByID(results[0].ID, map[string]any{
    "email": "alice@example.com",
    "role":  "admin",
})
```

### 2. Blog with full-text search

```go
posts := db.Model("posts").Schema(storage.ModelSchema{
    Fields: []storage.FieldSchema{
        {Name: "title",  Search: true},
        {Name: "body",   Search: true},
        {Name: "status", Index: true},
    },
})

// publish
_, _ = posts.Create(map[string]any{
    "title":  "Getting started with Go embedded databases",
    "body":   "In this tutorial we explore the store module...",
    "status": "published",
    "author": "alice",
})

// search — only published posts matching the query
all, _ := posts.Search("embedded database")
var published []storage.Document
for _, p := range all {
    if p.Data["status"] == "published" {
        published = append(published, p)
    }
}
```

### 3. Inventory management with atomic stock updates

Without a transaction, concurrent decrements can race:

```go
// WRONG — read-modify-write without a lock
doc, _ := inventory.FindByID(itemID)
newQty := doc.Data["qty"].(float64) - 1
_ = inventory.UpdateByID(itemID, map[string]any{"qty": newQty})
// Two goroutines reading the same qty will both write qty-1 instead of qty-2
```

Use a transaction to make the read-modify-write atomic:

```go
// CORRECT — all under the engine write lock
err := db.Tx(func(tx *storage.Tx) error {
    doc, err := db.Model("inventory").FindByID(itemID)
    if err != nil {
        return err
    }
    qty, _ := doc.Data["qty"].(float64)
    if qty <= 0 {
        return errors.New("out of stock")
    }
    return tx.Model("inventory").UpdateByID(itemID, map[string]any{
        "qty": qty - 1,
        "sku": doc.Data["sku"],
    })
})
```

> **Note:** `FindByID` inside the transaction callback reads the committed state at that moment. The update inside `Tx` is written atomically. If two goroutines call this concurrently, one will see the updated quantity of the other because `UpdateByID` inside the transaction holds the write lock during the commit phase.

### 4. Audit log with pre/post hooks

```go
db.Post("update", func(hc *storage.HookCtx) {
    _, _ = db.Model("audit_log").Create(map[string]any{
        "table":  hc.Table,
        "docID":  hc.ID,
        "action": "update",
        "data":   fmt.Sprintf("%v", hc.Data),
        "ts":     time.Now().Unix(),
    })
})

db.Post("delete", func(hc *storage.HookCtx) {
    _, _ = db.Model("audit_log").Create(map[string]any{
        "table":  hc.Table,
        "docID":  hc.ID,
        "action": "delete",
        "ts":     time.Now().Unix(),
    })
})
```

### 5. Cache invalidation via Watch

```go
var cache sync.Map // id → cached value

users := db.Model("users")
users.Watch(func(e storage.Event) {
    cache.Delete(e.Doc.ID)
})

// Subsequent FindByID calls after an update will miss the cache and re-fetch
```

### 6. Multi-step order processing with rollback

```go
err := db.Tx(func(tx *storage.Tx) error {
    // 1. reserve stock
    if err := tx.Model("inventory").UpdateByID(skuID, reservedData); err != nil {
        return err
    }
    // 2. create the order
    order, err := tx.Model("orders").Create(orderData)
    if err != nil {
        return err
    }
    // 3. create payment record
    _, err = tx.Model("payments").Create(map[string]any{
        "orderID": order.ID,
        "amount":  total,
        "status":  "pending",
    })
    return err // if any step fails, nothing is written
})
```

---

## Concurrency model

Every table (`Model`) has its own `*engine`. The engine holds a single `sync.RWMutex`:

- **Multiple concurrent readers** — `FindByID`, `Find().Exec()`, `Search()` all hold `RLock`. They run in parallel across different goroutines.
- **Exclusive writers** — `Create`, `UpdateByID`, `DeleteByID`, and transaction commits hold `WLock`. Writers never overlap with readers or other writers on the same table.
- **Different tables are fully independent** — concurrent writes to `"users"` and `"orders"` never block each other.

```
Goroutine A: FindByID("users", id1)  ─── RLock(users) ───────────────── RUnlock
Goroutine B: FindByID("users", id2)  ─── RLock(users) ───────────────── RUnlock
Goroutine C: Create("users", data)   ────────────────── WLock(users) ─── WUnlock
Goroutine D: Create("orders", data)  ─── WLock(orders) ─────────────────────────
```

Goroutines A and B run simultaneously. C waits until A and B finish. D runs concurrently with A, B, and C because it targets a different table.

---

## Race condition scenarios and how they are handled

### Scenario 1: Lost update (read-modify-write)

**Problem:** Two goroutines read a document, both compute an updated value, and both write — the first write is silently overwritten.

```go
// Goroutine 1              // Goroutine 2
doc, _ := m.FindByID(id)   doc, _ := m.FindByID(id)
qty := doc.Data["qty"]     qty := doc.Data["qty"]      // both read 10
newQty := qty.(float64)-1  newQty := qty.(float64)-1
m.UpdateByID(id, ...)      m.UpdateByID(id, ...)       // both write 9 — should be 8
```

**Resolution:** Wrap read-modify-write in `Tx`. The commit phase holds the write lock, so no other writer can interleave:

```go
db.Tx(func(tx *storage.Tx) error {
    doc, _ := db.Model("items").FindByID(id)
    qty := doc.Data["qty"].(float64)
    return tx.Model("items").UpdateByID(id, map[string]any{"qty": qty - 1})
})
```

The `FindByID` inside the closure reads under `RLock`. The `tx.commit()` acquires `WLock`, writes the update to disk, and updates the index — all before any other goroutine can read the new value.

### Scenario 2: Double-create (duplicate entry)

**Problem:** Two goroutines check that a username does not exist and both proceed to create — resulting in two documents with the same username.

```go
// Goroutine 1                           // Goroutine 2
docs, _ := m.Find().Where("username", ...).Exec()
// both see empty result
m.Create(map[string]any{"username": "alice"})
m.Create(map[string]any{"username": "alice"}) // duplicate
```

**Resolution:** Perform the check-and-create inside a transaction. Each `Create` inside the commit holds the table write lock, so the second create can be made conditional at the application level. For strict uniqueness, check inside the callback:

```go
db.Tx(func(tx *storage.Tx) error {
    existing, _ := db.Model("users").Find().Where("username", storage.OpEq, "alice").Limit(1).Exec()
    if len(existing) > 0 {
        return errors.New("username already taken")
    }
    _, err := tx.Model("users").Create(map[string]any{"username": "alice"})
    return err
})
```

Because `Find` holds `RLock` and `tx.commit` acquires `WLock`, no two goroutines can be inside the commit phase simultaneously on the same table.

### Scenario 3: Delete-then-read

**Problem:** Goroutine A deletes a document while goroutine B is about to read it.

**Resolution:** This is safe. `DeleteByID` holds `WLock` for the full duration (write to disk + index removal). Any goroutine waiting on `RLock` will see the document as absent after the write lock is released. `FindByID` returns `ErrNotFound` in this case — no partial state is visible.

### Scenario 4: Watcher fires during ongoing write

**Problem:** A Watch callback reads stale data because the engine lock was released before the event was delivered.

**Resolution:** The `notify` call in `Create`/`UpdateByID`/`DeleteByID` happens **after** the engine write lock is released and after the index is updated. The `Event.Doc` carries the fully committed document value. Watch callbacks therefore always see the post-write state.

### Scenario 5: Compaction during live traffic

**Problem:** Compaction rewrites segment files. Concurrent readers or writers might see missing or half-written files.

**Resolution:** `Compact` acquires the full engine `WLock` before touching any files and holds it for the entire duration. All reads and writes on the table block until compaction finishes. Segment files are written to a temporary `_compact/` directory first; they are only moved into place once fully written — so a crash during compaction leaves the original segments untouched.

### Scenario 6: Concurrent watcher registration and notification

**Problem:** A goroutine registers a watcher at the same time another goroutine triggers an event.

**Resolution:** `Watch` and `notify` use the model's own `sync.RWMutex` (separate from the engine lock). `Watch` holds the model `WLock` to append the watcher. `notify` holds the model `RLock` to copy the watcher slice. Multiple notifications can run concurrently; registrations wait only for the notify to finish copying.

---

## Atomic operations

### `Create` — atomic insert

A document either appears in the primary index with its complete payload, or it does not appear at all. The write sequence:

1. Generate ID (`crypto/rand` — no ID is ever reused).
2. Marshal payload.
3. Acquire `WLock`.
4. Append record to segment.
5. Update primary/secondary/text indexes.
6. Release `WLock`.

Step 4 is a single `os.File.Write` call. If it returns an error (e.g. disk full), the index is not updated and the partial record at the end of the segment will be skipped by the CRC32 check on the next replay.

### `UpdateByID` — atomic replace

The entire update (read old createdAt → write new record → update index) happens under a single `WLock`. Readers either see the old document or the new one — never a mix.

### `DeleteByID` — atomic removal

A `recDelete` record is appended under `WLock`, then the ID is removed from all indexes. After the lock is released, `FindByID` returns `ErrNotFound`. The old data on disk is not overwritten (append-only); it is removed logically from the index and physically during the next `Compact`.

### Transaction commit — atomic multi-table write

```
WLock(table1) → write TX_BEGIN → write ops → write TX_COMMIT → fsync → update index → WUnlock(table1)
WLock(table2) → write TX_BEGIN → write ops → write TX_COMMIT → fsync → update index → WUnlock(table2)
```

Each table's commit is independent. For cross-table transactions the commit is applied table by table; partial cross-table commits (table1 committed, crash before table2) are possible in theory. For strict cross-table atomicity, keep multi-table operations within a single table or accept the partial-commit risk for infrequent, compensatable operations.

### ID generation — collision-free

```go
b := make([]byte, 16) // 128 bits
rand.Read(b)
hex.EncodeToString(b) // 32-char lowercase hex
```

128 bits of entropy from `crypto/rand` makes ID collisions statistically impossible (probability < 10⁻³⁰ even with 10⁹ documents).

---

## Crash recovery

On `Open`, the engine scans all segment files in order from oldest to newest:

1. **Valid records** — replayed into the primary index. Each `recUpdate` overwrites the previous location for that ID. Each `recDelete` removes the ID.
2. **Corrupted tail record** — a partial write at the very end of the last segment (process killed mid-write) is detected by CRC32 mismatch and silently skipped. All preceding records are intact.
3. **Uncommitted transactions** — if a `TX_BEGIN` record exists without a matching `TX_COMMIT`, all records bearing that `txID` are discarded. The segment bytes remain on disk (append-only) but are never applied to the index.

No manual intervention is required after a crash. The database is always consistent from the last successfully committed operation.

---

## Storage layout

```
<path>/
├── users/
│   ├── seg_0000000.db    ← first segment, append-only
│   └── seg_0000001.db    ← second segment (after first reaches 64 MB)
├── orders/
│   └── seg_0000000.db
└── audit_log/
    └── seg_0000000.db
```

### Record wire format

Each record occupies a contiguous region:

```
[4 bytes] totalSize   — bytes following this field
[4 bytes] CRC32       — checksum of all bytes after this field
[1 byte ] recType     — 1=insert 2=update 3=delete 4=TX_BEGIN 5=TX_COMMIT 6=TX_ROLLBACK
[8 bytes] txID        — 0 for non-transactional writes
[2 bytes] tableLen; [N bytes] table name
[2 bytes] idLen;    [N bytes] document ID
[8 bytes] timestamp   — unix nanoseconds
[4 bytes] payloadLen; [N bytes] JSON payload
```

Segment rotation: a new segment file is opened automatically when the current one exceeds **64 MB**. The segment ID counter is monotonically increasing; files are named `seg_NNNNNNN.db`.

---

## Errors

| Error | When |
|-------|------|
| `ErrNotFound` | `FindByID`, `UpdateByID`, or `DeleteByID` called with an ID that does not exist |
| `ErrInvalidID` | Empty string passed as ID |
| `ErrInvalidTable` | Table name contains characters outside `[A-Za-z0-9_]` or is empty |
| `ErrCorrupt` | Record checksum mismatch when reading from disk |
| `ErrClosed` | Any operation after `Close()` |
| `ErrTxAborted` | Sentinel you can return from a `Tx` callback to signal an intentional rollback |
