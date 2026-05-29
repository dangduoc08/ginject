package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/testutils"
)

func tempDB(t *testing.T) (*DB, func()) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "testdb")
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return db, func() { _ = db.Close() }
}

// ---- record encoding ----

func TestEncodeDecodeRecord_RoundTrip(t *testing.T) {
	r := record{
		rtype:     recInsert,
		txID:      42,
		table:     "users",
		id:        "abc123",
		timestamp: time.Now().UnixNano(),
		payload:   []byte(`{"hello":"world"}`),
	}
	data := encodeRecord(r)
	got, n, err := decodeRecord(data)
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "decode must not error"))
	}
	if n != len(data) {
		t.Error(testutils.DiffMessage(n, len(data), "consumed bytes must equal record length"))
	}
	if got.rtype != r.rtype {
		t.Error(testutils.DiffMessage(got.rtype, r.rtype, "rtype"))
	}
	if got.txID != r.txID {
		t.Error(testutils.DiffMessage(got.txID, r.txID, "txID"))
	}
	if got.table != r.table {
		t.Error(testutils.DiffMessage(got.table, r.table, "table"))
	}
	if got.id != r.id {
		t.Error(testutils.DiffMessage(got.id, r.id, "id"))
	}
	if string(got.payload) != string(r.payload) {
		t.Error(testutils.DiffMessage(string(got.payload), string(r.payload), "payload"))
	}
}

func TestDecodeRecord_Corrupt_TruncatedInput(t *testing.T) {
	_, _, err := decodeRecord([]byte{1, 2, 3})
	if err != ErrCorrupt {
		t.Error(testutils.DiffMessage(err, ErrCorrupt, "short input must return ErrCorrupt"))
	}
}

func TestDecodeRecord_Corrupt_BadChecksum(t *testing.T) {
	r := record{rtype: recInsert, table: "t", id: "1", payload: []byte(`{}`)}
	data := encodeRecord(r)
	data[5] ^= 0xFF // corrupt checksum byte
	_, _, err := decodeRecord(data)
	if err != ErrCorrupt {
		t.Error(testutils.DiffMessage(err, ErrCorrupt, "bad checksum must return ErrCorrupt"))
	}
}

func TestEncodeDecodeRecord_EmptyPayload(t *testing.T) {
	r := record{rtype: recDelete, table: "t", id: "x"}
	data := encodeRecord(r)
	got, _, err := decodeRecord(data)
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "empty payload must decode"))
	}
	if len(got.payload) != 0 {
		t.Error(testutils.DiffMessage(len(got.payload), 0, "empty payload"))
	}
}

// ---- document payload ----

func TestMarshalUnmarshalPayload_RoundTrip(t *testing.T) {
	data := map[string]any{"name": "Alice", "age": float64(30)}
	createdAt := time.Now().Add(-time.Hour).Truncate(time.Nanosecond)
	updatedAt := time.Now().Truncate(time.Nanosecond)
	b, err := marshalPayload(data, createdAt, updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	gotData, gotCreated, gotUpdated, err := unmarshalPayload(b)
	if err != nil {
		t.Fatal(err)
	}
	if gotData["name"] != "Alice" {
		t.Error(testutils.DiffMessage(gotData["name"], "Alice", "name field"))
	}
	if !gotCreated.Equal(createdAt) {
		t.Error(testutils.DiffMessage(gotCreated, createdAt, "createdAt"))
	}
	if !gotUpdated.Equal(updatedAt) {
		t.Error(testutils.DiffMessage(gotUpdated, updatedAt, "updatedAt"))
	}
}

// ---- tokenizer ----

func TestTokenize_Basic(t *testing.T) {
	tokens := tokenize("Hello World")
	if len(tokens) != 2 {
		t.Error(testutils.DiffMessage(len(tokens), 2, "two tokens"))
	}
	if tokens[0] != "hello" {
		t.Error(testutils.DiffMessage(tokens[0], "hello", "lowercase"))
	}
}

func TestTokenize_ShortTokensDropped(t *testing.T) {
	tokens := tokenize("a bb ccc")
	// "a" (len=1) dropped, "bb" (len=2) kept, "ccc" kept
	for _, tok := range tokens {
		if len(tok) < 2 {
			t.Error(testutils.DiffMessage(tok, "(len>=2)", "short token must be dropped"))
		}
	}
}

func TestTokenize_Punctuation(t *testing.T) {
	tokens := tokenize("foo-bar,baz")
	if len(tokens) != 3 {
		t.Error(testutils.DiffMessage(len(tokens), 3, "punctuation splits tokens"))
	}
}

func TestTokenize_EmptyString(t *testing.T) {
	tokens := tokenize("")
	if len(tokens) != 0 {
		t.Error(testutils.DiffMessage(len(tokens), 0, "empty input"))
	}
}

// ---- validateTableName ----

func TestValidateTableName_Valid(t *testing.T) {
	cases := []string{"users", "blog_posts", "A1", "TABLE_123"}
	for _, name := range cases {
		if err := validateTableName(name); err != nil {
			t.Error(testutils.DiffMessage(err, nil, name+" must be valid"))
		}
	}
}

func TestValidateTableName_Invalid(t *testing.T) {
	cases := []string{"", "foo/bar", "../etc", "foo bar", "foo.bar"}
	for _, name := range cases {
		if err := validateTableName(name); err == nil {
			t.Error(testutils.DiffMessage(nil, ErrInvalidTable, name+" must be invalid"))
		}
	}
}

// ---- CRUD ----

func TestModel_Create_FindByID(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	doc, err := m.Create(map[string]any{"name": "Bob", "age": 25})
	if err != nil {
		t.Fatal(err)
	}
	if doc.ID == "" {
		t.Error(testutils.DiffMessage(doc.ID, "<non-empty>", "ID must be set"))
	}
	if doc.Data["name"] != "Bob" {
		t.Error(testutils.DiffMessage(doc.Data["name"], "Bob", "name"))
	}

	found, err := m.FindByID(doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != doc.ID {
		t.Error(testutils.DiffMessage(found.ID, doc.ID, "FindByID returns correct doc"))
	}
	if found.Data["name"] != "Bob" {
		t.Error(testutils.DiffMessage(found.Data["name"], "Bob", "name persisted"))
	}
}

func TestModel_FindByID_NotFound(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	_, err := db.Model("users").FindByID("nonexistent")
	if err != ErrNotFound {
		t.Error(testutils.DiffMessage(err, ErrNotFound, "missing id must return ErrNotFound"))
	}
}

func TestModel_FindByID_EmptyID(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	_, err := db.Model("users").FindByID("")
	if err != ErrInvalidID {
		t.Error(testutils.DiffMessage(err, ErrInvalidID, "empty id must return ErrInvalidID"))
	}
}

func TestModel_UpdateByID(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	doc, _ := m.Create(map[string]any{"name": "Alice"})
	if err := m.UpdateByID(doc.ID, map[string]any{"name": "Alice2"}); err != nil {
		t.Fatal(err)
	}
	updated, err := m.FindByID(doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Data["name"] != "Alice2" {
		t.Error(testutils.DiffMessage(updated.Data["name"], "Alice2", "update must persist"))
	}
	if !updated.CreatedAt.Equal(doc.CreatedAt) {
		t.Error(testutils.DiffMessage(updated.CreatedAt, doc.CreatedAt, "createdAt must not change on update"))
	}
}

func TestModel_UpdateByID_NotFound(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	err := db.Model("users").UpdateByID("ghost", map[string]any{"x": 1})
	if err != ErrNotFound {
		t.Error(testutils.DiffMessage(err, ErrNotFound, "update missing doc must error"))
	}
}

func TestModel_DeleteByID(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	doc, _ := m.Create(map[string]any{"name": "Delete Me"})
	if err := m.DeleteByID(doc.ID); err != nil {
		t.Fatal(err)
	}
	_, err := m.FindByID(doc.ID)
	if err != ErrNotFound {
		t.Error(testutils.DiffMessage(err, ErrNotFound, "deleted doc must not be found"))
	}
}

func TestModel_DeleteByID_NotFound(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	err := db.Model("users").DeleteByID("ghost")
	if err != ErrNotFound {
		t.Error(testutils.DiffMessage(err, ErrNotFound, "delete missing doc must error"))
	}
}

// ---- query ----

func TestQuery_Find_All(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("items")
	for i := 0; i < 5; i++ {
		_, _ = m.Create(map[string]any{"n": i})
	}
	docs, err := m.Find().Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 5 {
		t.Error(testutils.DiffMessage(len(docs), 5, "find all returns 5 docs"))
	}
}

func TestQuery_Find_WithLimit(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("items")
	for i := 0; i < 10; i++ {
		_, _ = m.Create(map[string]any{"n": i})
	}
	docs, _ := m.Find().Limit(3).Exec()
	if len(docs) != 3 {
		t.Error(testutils.DiffMessage(len(docs), 3, "limit 3"))
	}
}

func TestQuery_Find_WithSkip(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("items")
	for i := 0; i < 5; i++ {
		_, _ = m.Create(map[string]any{"n": i})
	}
	docs, _ := m.Find().Skip(3).Exec()
	if len(docs) != 2 {
		t.Error(testutils.DiffMessage(len(docs), 2, "skip 3 of 5"))
	}
}

func TestQuery_Where_EqSecondaryIndex(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})
	_, _ = m.Create(map[string]any{"role": "admin"})
	_, _ = m.Create(map[string]any{"role": "user"})
	_, _ = m.Create(map[string]any{"role": "admin"})

	docs, err := m.Find().Where("role", OpEq, "admin").Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Error(testutils.DiffMessage(len(docs), 2, "two admins"))
	}
}

func TestQuery_Where_EqFullScan(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	_, _ = m.Create(map[string]any{"role": "admin"})
	_, _ = m.Create(map[string]any{"role": "user"})

	docs, err := m.Find().Where("role", OpEq, "admin").Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Error(testutils.DiffMessage(len(docs), 1, "one admin via full scan"))
	}
}

func TestQuery_Where_Contains(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts")
	_, _ = m.Create(map[string]any{"title": "Hello World"})
	_, _ = m.Create(map[string]any{"title": "Goodbye"})

	docs, _ := m.Find().Where("title", OpContains, "Hello").Exec()
	if len(docs) != 1 {
		t.Error(testutils.DiffMessage(len(docs), 1, "contains match"))
	}
}

func TestQuery_Where_NoMatch(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	_, _ = m.Create(map[string]any{"role": "user"})

	docs, _ := m.Find().Where("role", OpEq, "admin").Exec()
	if len(docs) != 0 {
		t.Error(testutils.DiffMessage(len(docs), 0, "no match"))
	}
}

// ---- text search ----

func TestModel_Search(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "content", Search: true}},
	})
	_, _ = m.Create(map[string]any{"content": "golang embedded database"})
	_, _ = m.Create(map[string]any{"content": "python web framework"})
	_, _ = m.Create(map[string]any{"content": "golang web framework"})

	docs, err := m.Search("golang")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Error(testutils.DiffMessage(len(docs), 2, "two golang docs"))
	}
}

func TestModel_Search_MultiTermAND(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "content", Search: true}},
	})
	_, _ = m.Create(map[string]any{"content": "golang embedded database"})
	_, _ = m.Create(map[string]any{"content": "golang web server"})

	docs, _ := m.Search("golang embedded")
	if len(docs) != 1 {
		t.Error(testutils.DiffMessage(len(docs), 1, "AND semantics: only embedded golang"))
	}
}

func TestModel_Search_NoResults(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "content", Search: true}},
	})
	_, _ = m.Create(map[string]any{"content": "hello world"})
	docs, _ := m.Search("golang")
	if len(docs) != 0 {
		t.Error(testutils.DiffMessage(len(docs), 0, "no match"))
	}
}

// ---- transactions ----

func TestTx_Commit(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	var id1, id2 string
	err := db.Tx(func(tx *Tx) error {
		d1, err := tx.Model("users").Create(map[string]any{"name": "Alice"})
		if err != nil {
			return err
		}
		d2, err := tx.Model("users").Create(map[string]any{"name": "Bob"})
		if err != nil {
			return err
		}
		id1, id2 = d1.ID, d2.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	m := db.Model("users")
	if _, err := m.FindByID(id1); err != nil {
		t.Error(testutils.DiffMessage(err, nil, "doc1 must exist after tx commit"))
	}
	if _, err := m.FindByID(id2); err != nil {
		t.Error(testutils.DiffMessage(err, nil, "doc2 must exist after tx commit"))
	}
}

func TestTx_Rollback_OnError(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	var savedID string
	_ = db.Tx(func(tx *Tx) error {
		d, _ := tx.Model("users").Create(map[string]any{"name": "Alice"})
		savedID = d.ID
		return ErrTxAborted
	})

	// tx was rolled back — document should not be committed
	// Note: in our design, rollback means the fn returned an error and commit is skipped
	m := db.Model("users")
	_, err := m.FindByID(savedID)
	if err != ErrNotFound {
		t.Error(testutils.DiffMessage(err, ErrNotFound, "rolled-back doc must not be found"))
	}
}

func TestTx_EmptyCommit(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	err := db.Tx(func(tx *Tx) error { return nil })
	if err != nil {
		t.Error(testutils.DiffMessage(err, nil, "empty tx must not error"))
	}
}

// ---- persistence across Open ----

func TestPersistence_AfterReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	db1, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m1 := db1.Model("users")
	doc, _ := m1.Create(map[string]any{"name": "Persist"})
	_ = db1.Close()

	db2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db2.Close() }()
	m2 := db2.Model("users")
	found, err := m2.FindByID(doc.ID)
	if err != nil {
		t.Fatal(testutils.DiffMessage(err, nil, "doc must survive reopen"))
	}
	if found.Data["name"] != "Persist" {
		t.Error(testutils.DiffMessage(found.Data["name"], "Persist", "name persisted"))
	}
}

func TestPersistence_DeleteSurvivesReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	db1, _ := Open(dir)
	m1 := db1.Model("users")
	doc, _ := m1.Create(map[string]any{"name": "Gone"})
	_ = m1.DeleteByID(doc.ID)
	_ = db1.Close()

	db2, _ := Open(dir)
	defer func() { _ = db2.Close() }()
	_, err := db2.Model("users").FindByID(doc.ID)
	if err != ErrNotFound {
		t.Error(testutils.DiffMessage(err, ErrNotFound, "deleted doc must stay deleted after reopen"))
	}
}

// ---- compaction ----

func TestCompact_LiveRecordsPreserved(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	var ids []string
	for i := 0; i < 10; i++ {
		doc, _ := m.Create(map[string]any{"i": i})
		ids = append(ids, doc.ID)
	}
	// delete half
	for _, id := range ids[:5] {
		_ = m.DeleteByID(id)
	}

	if err := db.Compact(); err != nil {
		t.Fatal(err)
	}

	// deleted docs gone
	for _, id := range ids[:5] {
		if _, err := m.FindByID(id); err != ErrNotFound {
			t.Error(testutils.DiffMessage(err, ErrNotFound, "deleted doc must not exist post-compact"))
		}
	}
	// live docs still accessible
	for _, id := range ids[5:] {
		if _, err := m.FindByID(id); err != nil {
			t.Error(testutils.DiffMessage(err, nil, "live doc must survive compact"))
		}
	}
}

// ---- hooks ----

func TestHooks_Pre_Post(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	var preEvents, postEvents []string
	db.Pre("create", func(hc *HookCtx) { preEvents = append(preEvents, hc.Event) })
	db.Post("create", func(hc *HookCtx) { postEvents = append(postEvents, hc.Event) })

	_, _ = db.Model("users").Create(map[string]any{"x": 1})
	if len(preEvents) != 1 {
		t.Error(testutils.DiffMessage(len(preEvents), 1, "pre hook must fire"))
	}
	if len(postEvents) != 1 {
		t.Error(testutils.DiffMessage(len(postEvents), 1, "post hook must fire"))
	}
}

// ---- watch ----

func TestWatch_CreateEvent(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	var events []Event
	unsubscribe := m.Watch(func(e Event) { events = append(events, e) })
	defer unsubscribe()

	_, _ = m.Create(map[string]any{"name": "Alice"})
	if len(events) != 1 {
		t.Error(testutils.DiffMessage(len(events), 1, "create event must fire"))
	}
	if events[0].Type != EventCreate {
		t.Error(testutils.DiffMessage(events[0].Type, EventCreate, "event type"))
	}
}

func TestWatch_Unsubscribe(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	var count int
	unsub := m.Watch(func(e Event) { count++ })
	_, _ = m.Create(map[string]any{"x": 1})
	unsub()
	_, _ = m.Create(map[string]any{"x": 2})
	if count != 1 {
		t.Error(testutils.DiffMessage(count, 1, "unsubscribed watcher must not fire again"))
	}
}

// ---- flush & close ----

func TestFlush_NoError(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()
	_, _ = db.Model("users").Create(map[string]any{"x": 1})
	if err := db.Flush(); err != nil {
		t.Error(testutils.DiffMessage(err, nil, "Flush must not error"))
	}
}

func TestClose_ErrClosed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "db")
	db, _ := Open(dir)
	_ = db.Close()
	if err := db.Close(); err != ErrClosed {
		t.Error(testutils.DiffMessage(err, ErrClosed, "double close must return ErrClosed"))
	}
}

// ---- security: path traversal in table name ----

func TestModel_PathTraversal_Panics(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "path traversal must panic"))
		}
	}()
	_ = db.Model("../../etc/passwd")
}

func TestModel_EmptyTableName_Panics(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "empty table must panic"))
		}
	}()
	_ = db.Model("")
}

// ---- concurrency ----

func TestConcurrent_Creates(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_, _ = m.Create(map[string]any{"n": n})
		}(i)
	}
	wg.Wait()

	docs, err := m.Find().Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 100 {
		t.Error(testutils.DiffMessage(len(docs), 100, "all concurrent creates persisted"))
	}
}

func TestConcurrent_ReadWrite(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("items")
	doc, _ := m.Create(map[string]any{"v": 0})
	id := doc.ID

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			_ = m.UpdateByID(id, map[string]any{"v": n})
		}(i)
		go func() {
			defer wg.Done()
			_, _ = m.FindByID(id)
		}()
	}
	wg.Wait()
}

// ---- segment file existence ----

func TestSegmentFile_Created(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "db")
	db, _ := Open(dir)
	_, _ = db.Model("users").Create(map[string]any{"x": 1})
	_ = db.Close()

	entries, err := os.ReadDir(filepath.Join(dir, "users"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "seg_0000000.db" {
			found = true
			break
		}
	}
	if !found {
		t.Error(testutils.DiffMessage(found, true, "seg_0000000.db must exist"))
	}
}

// ---- index update after secondary index schema ----

func TestSecondaryIndex_UpdateRemovesOldEntry(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})
	doc, _ := m.Create(map[string]any{"role": "user"})
	_ = m.UpdateByID(doc.ID, map[string]any{"role": "admin"})

	// old "user" entry must be gone
	users, _ := m.Find().Where("role", OpEq, "user").Exec()
	if len(users) != 0 {
		t.Error(testutils.DiffMessage(len(users), 0, "old role entry must be removed"))
	}
	admins, _ := m.Find().Where("role", OpEq, "admin").Exec()
	if len(admins) != 1 {
		t.Error(testutils.DiffMessage(len(admins), 1, "new role entry must exist"))
	}
}

func TestTextIndex_DeleteRemovesTerms(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "body", Search: true}},
	})
	doc, _ := m.Create(map[string]any{"body": "golang rocks"})
	_ = m.DeleteByID(doc.ID)

	results, _ := m.Search("golang")
	if len(results) != 0 {
		t.Error(testutils.DiffMessage(len(results), 0, "deleted doc must not appear in search"))
	}
}
