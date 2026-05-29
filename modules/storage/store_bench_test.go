package storage

import (
	"fmt"
	"testing"
)

func benchDB(b *testing.B) (*DB, func()) {
	b.Helper()
	dir := b.TempDir()
	db, err := Open(dir)
	if err != nil {
		b.Fatal(err)
	}
	return db, func() { _ = db.Close() }
}

func BenchmarkCreate(b *testing.B) {
	db, cleanup := benchDB(b)
	defer cleanup()
	m := db.Model("users")
	b.ResetTimer()
	for range b.N {
		_, _ = m.Create(map[string]any{"name": "Alice", "role": "user"})
	}
}

func BenchmarkFindByID(b *testing.B) {
	db, cleanup := benchDB(b)
	defer cleanup()
	m := db.Model("users")
	doc, _ := m.Create(map[string]any{"name": "Bob"})
	id := doc.ID
	b.ResetTimer()
	for range b.N {
		_, _ = m.FindByID(id)
	}
}

func BenchmarkUpdateByID(b *testing.B) {
	db, cleanup := benchDB(b)
	defer cleanup()
	m := db.Model("users")
	doc, _ := m.Create(map[string]any{"name": "Bob"})
	id := doc.ID
	b.ResetTimer()
	for range b.N {
		_ = m.UpdateByID(id, map[string]any{"name": "Bob2"})
	}
}

func BenchmarkFind_NoIndex_100Docs(b *testing.B) {
	db, cleanup := benchDB(b)
	defer cleanup()
	m := db.Model("items")
	for i := 0; i < 100; i++ {
		_, _ = m.Create(map[string]any{"n": i, "role": "user"})
	}
	b.ResetTimer()
	for range b.N {
		_, _ = m.Find().Where("role", OpEq, "user").Exec()
	}
}

func BenchmarkFind_SecondaryIndex_1000Docs(b *testing.B) {
	db, cleanup := benchDB(b)
	defer cleanup()
	m := db.Model("items").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "role", Index: true}},
	})
	for i := 0; i < 1000; i++ {
		role := "user"
		if i%10 == 0 {
			role = "admin"
		}
		_, _ = m.Create(map[string]any{"n": i, "role": role})
	}
	b.ResetTimer()
	for range b.N {
		_, _ = m.Find().Where("role", OpEq, "admin").Exec()
	}
}

func BenchmarkSearch_1000Docs(b *testing.B) {
	db, cleanup := benchDB(b)
	defer cleanup()
	m := db.Model("posts").Schema(ModelSchema{
		Fields: []FieldSchema{{Name: "content", Search: true}},
	})
	words := []string{"golang", "database", "embedded", "performance", "concurrent"}
	for i := 0; i < 1000; i++ {
		w := words[i%len(words)]
		_, _ = m.Create(map[string]any{"content": fmt.Sprintf("%s article number %d", w, i)})
	}
	b.ResetTimer()
	for range b.N {
		_, _ = m.Search("golang")
	}
}

func BenchmarkTx_2Inserts(b *testing.B) {
	db, cleanup := benchDB(b)
	defer cleanup()
	b.ResetTimer()
	for range b.N {
		_ = db.Tx(func(tx *Tx) error {
			_, err := tx.Model("users").Create(map[string]any{"name": "Alice"})
			if err != nil {
				return err
			}
			_, err = tx.Model("users").Create(map[string]any{"name": "Bob"})
			return err
		})
	}
}

func BenchmarkEncodeRecord(b *testing.B) {
	r := record{
		rtype:     recInsert,
		table:     "users",
		id:        "abc123def456abc123def456abc12345",
		timestamp: 1234567890,
		payload:   []byte(`{"name":"Alice","role":"admin","age":30}`),
	}
	b.ResetTimer()
	for range b.N {
		_ = encodeRecord(r)
	}
}

func BenchmarkDecodeRecord(b *testing.B) {
	r := record{
		rtype:     recInsert,
		table:     "users",
		id:        "abc123def456abc123def456abc12345",
		timestamp: 1234567890,
		payload:   []byte(`{"name":"Alice","role":"admin","age":30}`),
	}
	data := encodeRecord(r)
	b.ResetTimer()
	for range b.N {
		_, _, _ = decodeRecord(data)
	}
}

func BenchmarkTokenize(b *testing.B) {
	s := "the quick brown fox jumps over the lazy dog and the embedded golang database"
	b.ResetTimer()
	for range b.N {
		_ = tokenize(s)
	}
}
