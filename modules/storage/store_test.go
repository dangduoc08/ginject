package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/internal/test"
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
		t.Error(test.DiffMessage(err, nil, "decode must not error"))
	}
	if n != len(data) {
		t.Error(test.DiffMessage(n, len(data), "consumed bytes must equal record length"))
	}
	if got.rtype != r.rtype {
		t.Error(test.DiffMessage(got.rtype, r.rtype, "rtype"))
	}
	if got.txID != r.txID {
		t.Error(test.DiffMessage(got.txID, r.txID, "txID"))
	}
	if got.table != r.table {
		t.Error(test.DiffMessage(got.table, r.table, "table"))
	}
	if got.id != r.id {
		t.Error(test.DiffMessage(got.id, r.id, "id"))
	}
	if string(got.payload) != string(r.payload) {
		t.Error(test.DiffMessage(string(got.payload), string(r.payload), "payload"))
	}
}

func TestDecodeRecord_Corrupt_TruncatedInput(t *testing.T) {
	_, _, err := decodeRecord([]byte{1, 2, 3})
	if err != ErrCorrupt {
		t.Error(test.DiffMessage(err, ErrCorrupt, "short input must return ErrCorrupt"))
	}
}

func TestDecodeRecord_Corrupt_BadChecksum(t *testing.T) {
	r := record{rtype: recInsert, table: "t", id: "1", payload: []byte(`{}`)}
	data := encodeRecord(r)
	data[5] ^= 0xFF // corrupt checksum byte
	_, _, err := decodeRecord(data)
	if err != ErrCorrupt {
		t.Error(test.DiffMessage(err, ErrCorrupt, "bad checksum must return ErrCorrupt"))
	}
}

func TestEncodeDecodeRecord_EmptyPayload(t *testing.T) {
	r := record{rtype: recDelete, table: "t", id: "x"}
	data := encodeRecord(r)
	got, _, err := decodeRecord(data)
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "empty payload must decode"))
	}
	if len(got.payload) != 0 {
		t.Error(test.DiffMessage(len(got.payload), 0, "empty payload"))
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
		t.Error(test.DiffMessage(gotData["name"], "Alice", "name field"))
	}
	if !gotCreated.Equal(createdAt) {
		t.Error(test.DiffMessage(gotCreated, createdAt, "createdAt"))
	}
	if !gotUpdated.Equal(updatedAt) {
		t.Error(test.DiffMessage(gotUpdated, updatedAt, "updatedAt"))
	}
}

// ---- tokenizer ----

func TestTokenize_Basic(t *testing.T) {
	tokens := tokenize("Hello World")
	if len(tokens) != 2 {
		t.Error(test.DiffMessage(len(tokens), 2, "two tokens"))
	}
	if tokens[0] != "hello" {
		t.Error(test.DiffMessage(tokens[0], "hello", "lowercase"))
	}
}

func TestTokenize_ShortTokensDropped(t *testing.T) {
	tokens := tokenize("a bb ccc")
	// "a" (len=1) dropped, "bb" (len=2) kept, "ccc" kept
	for _, tok := range tokens {
		if len(tok) < 2 {
			t.Error(test.DiffMessage(tok, "(len>=2)", "short token must be dropped"))
		}
	}
}

func TestTokenize_Punctuation(t *testing.T) {
	tokens := tokenize("foo-bar,baz")
	if len(tokens) != 3 {
		t.Error(test.DiffMessage(len(tokens), 3, "punctuation splits tokens"))
	}
}

func TestTokenize_EmptyString(t *testing.T) {
	tokens := tokenize("")
	if len(tokens) != 0 {
		t.Error(test.DiffMessage(len(tokens), 0, "empty input"))
	}
}

// ---- validateTableName ----

func TestValidateTableName_Valid(t *testing.T) {
	cases := []string{"users", "blog_posts", "A1", "TABLE_123"}
	for _, name := range cases {
		if err := validateTableName(name); err != nil {
			t.Error(test.DiffMessage(err, nil, name+" must be valid"))
		}
	}
}

func TestValidateTableName_Invalid(t *testing.T) {
	cases := []string{"", "foo/bar", "../etc", "foo bar", "foo.bar"}
	for _, name := range cases {
		if err := validateTableName(name); err == nil {
			t.Error(test.DiffMessage(nil, ErrInvalidTable, name+" must be invalid"))
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
		t.Error(test.DiffMessage(doc.ID, "<non-empty>", "ID must be set"))
	}
	if doc.Data["name"] != "Bob" {
		t.Error(test.DiffMessage(doc.Data["name"], "Bob", "name"))
	}

	found, err := m.FindByID(doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.ID != doc.ID {
		t.Error(test.DiffMessage(found.ID, doc.ID, "FindByID returns correct doc"))
	}
	if found.Data["name"] != "Bob" {
		t.Error(test.DiffMessage(found.Data["name"], "Bob", "name persisted"))
	}
}

func TestModel_FindByID_NotFound(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	_, err := db.Model("users").FindByID("nonexistent")
	if err != ErrNotFound {
		t.Error(test.DiffMessage(err, ErrNotFound, "missing id must return ErrNotFound"))
	}
}

func TestModel_FindByID_EmptyID(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	_, err := db.Model("users").FindByID("")
	if err != ErrInvalidID {
		t.Error(test.DiffMessage(err, ErrInvalidID, "empty id must return ErrInvalidID"))
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
		t.Error(test.DiffMessage(updated.Data["name"], "Alice2", "update must persist"))
	}
	if !updated.CreatedAt.Equal(doc.CreatedAt) {
		t.Error(test.DiffMessage(updated.CreatedAt, doc.CreatedAt, "createdAt must not change on update"))
	}
}

func TestModel_UpdateByID_NotFound(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	err := db.Model("users").UpdateByID("ghost", map[string]any{"x": 1})
	if err != ErrNotFound {
		t.Error(test.DiffMessage(err, ErrNotFound, "update missing doc must error"))
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
		t.Error(test.DiffMessage(err, ErrNotFound, "deleted doc must not be found"))
	}
}

func TestModel_DeleteByID_NotFound(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	err := db.Model("users").DeleteByID("ghost")
	if err != ErrNotFound {
		t.Error(test.DiffMessage(err, ErrNotFound, "delete missing doc must error"))
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
		t.Error(test.DiffMessage(len(docs), 5, "find all returns 5 docs"))
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
		t.Error(test.DiffMessage(len(docs), 3, "limit 3"))
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
		t.Error(test.DiffMessage(len(docs), 2, "skip 3 of 5"))
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
		t.Error(test.DiffMessage(len(docs), 2, "two admins"))
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
		t.Error(test.DiffMessage(len(docs), 1, "one admin via full scan"))
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
		t.Error(test.DiffMessage(len(docs), 1, "contains match"))
	}
}

func TestQuery_Where_NoMatch(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users")
	_, _ = m.Create(map[string]any{"role": "user"})

	docs, _ := m.Find().Where("role", OpEq, "admin").Exec()
	if len(docs) != 0 {
		t.Error(test.DiffMessage(len(docs), 0, "no match"))
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
		t.Error(test.DiffMessage(len(docs), 2, "two golang docs"))
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
		t.Error(test.DiffMessage(len(docs), 1, "AND semantics: only embedded golang"))
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
		t.Error(test.DiffMessage(len(docs), 0, "no match"))
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
		t.Error(test.DiffMessage(err, nil, "doc1 must exist after tx commit"))
	}
	if _, err := m.FindByID(id2); err != nil {
		t.Error(test.DiffMessage(err, nil, "doc2 must exist after tx commit"))
	}
}

func TestTx_IndexesMaintained_InsertUpdateDelete(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}, {Name: "bio", Search: true}},
	})

	var id string
	if err := db.Tx(func(tx *Tx) error {
		doc, err := tx.Model("users").Create(map[string]any{"role": "user", "bio": "writes golang"})
		if err != nil {
			return err
		}
		id = doc.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	users, _ := m.Find().Where("role", OpEq, "user").Exec()
	if len(users) != 1 {
		t.Error(test.DiffMessage(len(users), 1, "tx insert must populate the secondary index"))
	}
	hits, _ := m.Search("golang")
	if len(hits) != 1 {
		t.Error(test.DiffMessage(len(hits), 1, "tx insert must populate the text index"))
	}

	if err := db.Tx(func(tx *Tx) error {
		return tx.Model("users").UpdateByID(id, map[string]any{"role": "admin", "bio": "writes rust"})
	}); err != nil {
		t.Fatal(err)
	}

	stale, _ := m.Find().Where("role", OpEq, "user").Exec()
	if len(stale) != 0 {
		t.Error(test.DiffMessage(len(stale), 0, "tx update must drop the old secondary entry"))
	}
	admins, _ := m.Find().Where("role", OpEq, "admin").Exec()
	if len(admins) != 1 {
		t.Error(test.DiffMessage(len(admins), 1, "tx update must add the new secondary entry"))
	}
	staleTerms, _ := m.Search("golang")
	if len(staleTerms) != 0 {
		t.Error(test.DiffMessage(len(staleTerms), 0, "tx update must drop the old text terms"))
	}
	newTerms, _ := m.Search("rust")
	if len(newTerms) != 1 {
		t.Error(test.DiffMessage(len(newTerms), 1, "tx update must add the new text terms"))
	}

	if err := db.Tx(func(tx *Tx) error {
		return tx.Model("users").DeleteByID(id)
	}); err != nil {
		t.Fatal(err)
	}

	gone, _ := m.Find().Where("role", OpEq, "admin").Exec()
	if len(gone) != 0 {
		t.Error(test.DiffMessage(len(gone), 0, "tx delete must drop the secondary entry"))
	}
	goneTerms, _ := m.Search("rust")
	if len(goneTerms) != 0 {
		t.Error(test.DiffMessage(len(goneTerms), 0, "tx delete must drop the text terms"))
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
		t.Error(test.DiffMessage(err, ErrNotFound, "rolled-back doc must not be found"))
	}
}

func TestTx_EmptyCommit(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	err := db.Tx(func(tx *Tx) error { return nil })
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "empty tx must not error"))
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
		t.Fatal(test.DiffMessage(err, nil, "doc must survive reopen"))
	}
	if found.Data["name"] != "Persist" {
		t.Error(test.DiffMessage(found.Data["name"], "Persist", "name persisted"))
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
		t.Error(test.DiffMessage(err, ErrNotFound, "deleted doc must stay deleted after reopen"))
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
			t.Error(test.DiffMessage(err, ErrNotFound, "deleted doc must not exist post-compact"))
		}
	}
	// live docs still accessible
	for _, id := range ids[5:] {
		if _, err := m.FindByID(id); err != nil {
			t.Error(test.DiffMessage(err, nil, "live doc must survive compact"))
		}
	}
}

func TestCompact_SecondaryIndexPreserved_WithSchema(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})
	_, _ = m.Create(map[string]any{"role": "admin"})
	_, _ = m.Create(map[string]any{"role": "user"})
	doomed, _ := m.Create(map[string]any{"role": "admin"})
	_ = m.DeleteByID(doomed.ID)

	if err := db.Compact(); err != nil {
		t.Fatal(err)
	}

	eng, err := db.getEngine("users")
	if err != nil {
		t.Fatal(err)
	}
	if !eng.idx.hasSecondaryField("role") {
		t.Error(test.DiffMessage(false, true, "compaction must not discard the registered schema"))
	}
	if len(eng.idx.secondaryByField["role"]) == 0 {
		t.Error(test.DiffMessage(0, 1, "compaction must rebuild the secondary index for a schema that was already registered"))
	}

	docs, err := m.Find().Where("role", OpEq, "admin").Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Error(test.DiffMessage(len(docs), 1, "indexed lookup must return only live documents after compaction"))
	}
}

func TestCompact_TextIndexPreserved_WithSchema(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "title", Search: true}},
	})
	_, _ = m.Create(map[string]any{"title": "golang storage engine"})
	doomed, _ := m.Create(map[string]any{"title": "golang removed entry"})
	_ = m.DeleteByID(doomed.ID)

	if err := db.Compact(); err != nil {
		t.Fatal(err)
	}

	docs, err := m.Search("golang")
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Error(test.DiffMessage(len(docs), 1, "compaction must rebuild the text index for a schema that was already registered"))
	}
}

func TestPersistence_PrimaryIndexCompleteAfterReopenWithoutSchema(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	db1, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m1 := db1.Model("users")
	var ids []string
	for i := 0; i < 20; i++ {
		doc, err := m1.Create(map[string]any{"role": "admin", "i": i})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, doc.ID)
	}
	_ = db1.Close()

	db2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db2.Close() }()

	m2 := db2.Model("users")
	for _, id := range ids {
		if _, err := m2.FindByID(id); err != nil {
			t.Fatal(test.DiffMessage(err, nil, "every document must stay in the primary index after a reopen with no schema registered"))
		}
	}
}

func TestPersistence_SecondaryIndexRebuiltBySchemaAfterReopen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "persist")
	db1, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m1 := db1.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}, {Name: "bio", Search: true}},
	})
	_, _ = m1.Create(map[string]any{"role": "admin", "bio": "builds storage engines"})
	_, _ = m1.Create(map[string]any{"role": "user", "bio": "writes documentation"})
	_, _ = m1.Create(map[string]any{"role": "admin", "bio": "reviews storage patches"})
	_ = db1.Close()

	db2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db2.Close() }()

	m2 := db2.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}, {Name: "bio", Search: true}},
	})

	docs, err := m2.Find().Where("role", OpEq, "admin").Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Error(test.DiffMessage(len(docs), 2, "Schema must rebuild the secondary index from data loaded by a previous process"))
	}

	found, err := m2.Search("storage")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Error(test.DiffMessage(len(found), 2, "Schema must rebuild the text index from data loaded by a previous process"))
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
		t.Error(test.DiffMessage(len(preEvents), 1, "pre hook must fire"))
	}
	if len(postEvents) != 1 {
		t.Error(test.DiffMessage(len(postEvents), 1, "post hook must fire"))
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
		t.Error(test.DiffMessage(len(events), 1, "create event must fire"))
	}
	if events[0].Type != EventCreate {
		t.Error(test.DiffMessage(events[0].Type, EventCreate, "event type"))
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
		t.Error(test.DiffMessage(count, 1, "unsubscribed watcher must not fire again"))
	}
}

// ---- flush & close ----

func TestFlush_NoError(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()
	_, _ = db.Model("users").Create(map[string]any{"x": 1})
	if err := db.Flush(); err != nil {
		t.Error(test.DiffMessage(err, nil, "Flush must not error"))
	}
}

func TestClose_ErrClosed(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "db")
	db, _ := Open(dir)
	_ = db.Close()
	if err := db.Close(); err != ErrClosed {
		t.Error(test.DiffMessage(err, ErrClosed, "double close must return ErrClosed"))
	}
}

// ---- security: path traversal in table name ----

func TestModel_PathTraversal_Panics(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "path traversal must panic"))
		}
	}()
	_ = db.Model("../../etc/passwd")
}

func TestModel_EmptyTableName_Panics(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "empty table must panic"))
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
		t.Error(test.DiffMessage(len(docs), 100, "all concurrent creates persisted"))
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
		t.Error(test.DiffMessage(found, true, "seg_0000000.db must exist"))
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
		t.Error(test.DiffMessage(len(users), 0, "old role entry must be removed"))
	}
	admins, _ := m.Find().Where("role", OpEq, "admin").Exec()
	if len(admins) != 1 {
		t.Error(test.DiffMessage(len(admins), 1, "new role entry must exist"))
	}
}

func TestSchema_ClosedDB_Panics(t *testing.T) {
	db, cleanup := tempDB(t)
	cleanup()

	m := db.Model("users")
	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal(test.DiffMessage(nil, ErrClosed, "Schema must not silently skip indexing when the engine cannot be opened"))
		}
		if err, ok := rec.(error); !ok || err != ErrClosed {
			t.Error(test.DiffMessage(rec, ErrClosed, "Schema must panic with the underlying engine error"))
		}
	}()

	m.Schema(ModelSchema{Fields: []FieldSchema{{Name: "role", Index: true}}})
}

func TestSchema_IdenticalCall_SkipsRebuild(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})
	_, _ = m.Create(map[string]any{"role": "admin"})

	eng, err := db.getEngine("users")
	if err != nil {
		t.Fatal(err)
	}
	if len(eng.idx.secondaryByField["role"]) == 0 {
		t.Fatal(test.DiffMessage(0, 1, "first Schema call must build the secondary index"))
	}

	eng.mu.Lock()
	eng.idx.secondaryByField = make(map[string]map[string]map[string]bool)
	eng.mu.Unlock()

	m.Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})

	if len(eng.idx.secondaryByField) != 0 {
		t.Error(test.DiffMessage(len(eng.idx.secondaryByField), 0, "an identical Schema call must not re-scan the table"))
	}
}

func TestSchema_DuplicateFieldEntries_StillSkipsRebuild(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	dup := ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}, {Name: "role", Index: true}},
	}

	m := db.Model("users").Schema(dup)
	_, _ = m.Create(map[string]any{"role": "admin"})

	eng, err := db.getEngine("users")
	if err != nil {
		t.Fatal(err)
	}

	eng.mu.Lock()
	eng.idx.secondaryByField = make(map[string]map[string]map[string]bool)
	eng.mu.Unlock()

	m.Schema(dup)

	if len(eng.idx.secondaryByField) != 0 {
		t.Error(test.DiffMessage(len(eng.idx.secondaryByField), 0, "repeated field entries must not defeat the unchanged-schema check"))
	}
}

func TestSchema_SameFieldDifferentRole_Rebuilds(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "title", Index: true}},
	})
	_, _ = m.Create(map[string]any{"title": "golang storage"})

	m.Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "title", Search: true}},
	})

	found, err := m.Search("golang")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Error(test.DiffMessage(len(found), 1, "moving a field from Index to Search must rebuild the text index"))
	}

	eng, err := db.getEngine("posts")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := eng.idx.secondaryByField["title"]; ok {
		t.Error(test.DiffMessage(true, false, "moving a field from Index to Search must release its secondary index"))
	}
}

func TestSchema_Change_DropsStaleTextIndex(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "bio", Search: true}},
	})
	_, _ = m.Create(map[string]any{"bio": "builds storage engines", "role": "admin"})

	found, _ := m.Search("storage")
	if len(found) != 1 {
		t.Fatal(test.DiffMessage(len(found), 1, "text index must work while the search field is registered"))
	}

	m.Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})

	stale, err := m.Search("storage")
	if err != nil {
		t.Fatal(err)
	}
	if len(stale) != 0 {
		t.Error(test.DiffMessage(len(stale), 0, "a schema that no longer declares a search field must not keep answering Search from the old text index"))
	}

	eng, err := db.getEngine("posts")
	if err != nil {
		t.Fatal(err)
	}
	if len(eng.idx.idsByTerm) != 0 || len(eng.idx.termsByID) != 0 {
		t.Error(test.DiffMessage([]int{len(eng.idx.idsByTerm), len(eng.idx.termsByID)}, []int{0, 0}, "dropping a search field must release its text index"))
	}
}

func TestSchema_Change_DropsStaleSecondaryIndex(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})
	_, _ = m.Create(map[string]any{"role": "admin", "team": "core"})

	m.Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "team", Index: true}},
	})

	eng, err := db.getEngine("users")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := eng.idx.secondaryByField["role"]; ok {
		t.Error(test.DiffMessage(true, false, "dropping an indexed field must release its secondary index"))
	}
	if len(eng.idx.secondaryByField["team"]) == 0 {
		t.Error(test.DiffMessage(0, 1, "the newly indexed field must be populated"))
	}

	docs, err := m.Find().Where("role", OpEq, "admin").Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Error(test.DiffMessage(len(docs), 1, "a no-longer-indexed field must still be queryable by full scan"))
	}
}

func TestSecondaryIndex_EmptyFieldBucketReleased(t *testing.T) {
	db, cleanup := tempDB(t)
	defer cleanup()

	m := db.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})
	doc, _ := m.Create(map[string]any{"role": "admin"})
	_ = m.DeleteByID(doc.ID)

	eng, err := db.getEngine("users")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := eng.idx.secondaryByField["role"]; ok {
		t.Error(test.DiffMessage(true, false, "a field bucket must be released once its last value is gone"))
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
		t.Error(test.DiffMessage(len(results), 0, "deleted doc must not appear in search"))
	}
}

func TestOpenWithSchemas_BuildsIndexesInOnePass(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "declared")

	db1, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	m1 := db1.Model("users").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}, {Name: "bio", Search: true}},
	})
	_, _ = m1.Create(map[string]any{"role": "admin", "bio": "builds storage engines"})
	_, _ = m1.Create(map[string]any{"role": "user", "bio": "writes documentation"})
	_ = db1.Close()

	schemas := map[string]ModelSchema{
		"users": {Fields: []FieldSchema{{Name: "role", Index: true}, {Name: "bio", Search: true}}},
	}
	db2, err := OpenWithSchemas(dir, schemas)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db2.Close() }()

	m2 := db2.Model("users")

	docs, err := m2.Find().Where("role", OpEq, "admin").Exec()
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Error(test.DiffMessage(len(docs), 1, "a pre-declared schema must leave the secondary index ready without calling Schema"))
	}

	found, err := m2.Search("storage")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 {
		t.Error(test.DiffMessage(len(found), 1, "a pre-declared schema must leave the text index ready without calling Schema"))
	}

	eng, err := db2.getEngine("users")
	if err != nil {
		t.Fatal(err)
	}
	if !eng.idx.schemaEquals([]string{"role"}, []string{"bio"}) {
		t.Error(test.DiffMessage(false, true, "the engine must already carry the declared schema"))
	}

	eng.mu.Lock()
	eng.idx.secondaryByField = make(map[string]map[string]map[string]bool)
	eng.mu.Unlock()

	m2.Schema(schemas["users"])

	if len(eng.idx.secondaryByField) != 0 {
		t.Error(test.DiffMessage(len(eng.idx.secondaryByField), 0, "calling Schema with the already-declared fields must not trigger a second scan"))
	}
}

func TestOpenWithSchemas_NilSchemas_BehavesLikeOpen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nodeclared")

	db, err := OpenWithSchemas(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	m := db.Model("users")
	doc, err := m.Create(map[string]any{"role": "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.FindByID(doc.ID); err != nil {
		t.Error(test.DiffMessage(err, nil, "a database opened without declared schemas must still work"))
	}
}

func TestOpenWithSchemas_CopiesCallerMap(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "copied")

	schemas := map[string]ModelSchema{
		"users": {Fields: []FieldSchema{{Name: "role", Index: true}}},
	}
	db, err := OpenWithSchemas(dir, schemas)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()

	delete(schemas, "users")

	eng, err := db.getEngine("users")
	if err != nil {
		t.Fatal(err)
	}
	if !eng.idx.hasSecondaryField("role") {
		t.Error(test.DiffMessage(false, true, "mutating the caller's map after Open must not change the database's schemas"))
	}
}
