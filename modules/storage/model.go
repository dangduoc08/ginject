package storage

import (
	"sync"
	"sync/atomic"
	"time"
)

// EventType describes what happened to a document.
type EventType string

const (
	EventCreate EventType = "create"
	EventUpdate EventType = "update"
	EventDelete EventType = "delete"
)

// Event is emitted to watchers after a document changes.
type Event struct {
	Type EventType
	Doc  Document
}

// FieldSchema describes one field's indexing hints.
type FieldSchema struct {
	Name   string
	Index  bool // build secondary index on this field
	Search bool // build text search index on this field
}

// ModelSchema declares which fields should be indexed.
type ModelSchema struct {
	Fields []FieldSchema
}

// HookCtx carries context for pre/post hooks.
type HookCtx struct {
	Event string
	Table string
	ID    string
	Data  map[string]any
}

type hookFn func(*HookCtx)

type hookSet struct {
	mu   sync.RWMutex
	pre  map[string][]hookFn
	post map[string][]hookFn
}

func newHookSet() *hookSet {
	return &hookSet{
		pre:  make(map[string][]hookFn),
		post: make(map[string][]hookFn),
	}
}

var watcherCounter uint64

type watcher struct {
	id uint64
	fn func(Event)
}

// Model is the per-table API: Create, FindByID, UpdateByID, DeleteByID, Find, Search, Watch, Schema.
type Model struct {
	db       *DB
	table    string
	mu       sync.RWMutex
	watchers []*watcher
}

func newModel(db *DB, table string) *Model {
	return &Model{db: db, table: table}
}

// Schema registers field-level indexing hints and rebuilds secondary/text indexes
// from existing data. Must be called before requests arrive for accurate queries.
func (m *Model) Schema(s ModelSchema) *Model {
	var indexed, search []string
	for _, f := range s.Fields {
		if f.Index {
			indexed = append(indexed, f.Name)
		}
		if f.Search {
			search = append(search, f.Name)
		}
	}

	eng, err := m.db.getEngine(m.table)
	if err != nil {
		return m
	}
	eng.mu.Lock()
	eng.idx.setSchema(indexed, search)
	// rebuild secondary + text from primary index
	for id, loc := range eng.idx.primary {
		seg := eng.segByID(loc.segID)
		if seg == nil {
			continue
		}
		r, err := readRecordAt(seg, loc)
		if err != nil {
			continue
		}
		data, _, _, err := unmarshalPayload(r.payload)
		if err != nil {
			continue
		}
		eng.idx.updateSecondary(id, nil, data)
		eng.idx.updateText(id, nil, data)
	}
	eng.mu.Unlock()
	return m
}

// Create inserts a new document and returns it with a generated ID.
func (m *Model) Create(data map[string]any) (Document, error) {
	m.db.runHook(m.db.hooks, "pre", "create", m.table, "", data)

	id, err := newID()
	if err != nil {
		return Document{}, err
	}
	now := time.Now()
	payload, err := marshalPayload(data, now, now)
	if err != nil {
		return Document{}, err
	}
	r := record{
		rtype:     recInsert,
		table:     m.table,
		id:        id,
		timestamp: now.UnixNano(),
		payload:   payload,
	}

	eng, err := m.db.getEngine(m.table)
	if err != nil {
		return Document{}, err
	}
	eng.mu.Lock()
	loc, err := eng.writeRecord(r)
	if err != nil {
		eng.mu.Unlock()
		return Document{}, err
	}
	eng.idx.setPrimary(id, loc)
	eng.idx.updateSecondary(id, nil, data)
	eng.idx.updateText(id, nil, data)
	eng.mu.Unlock()

	doc := Document{ID: id, Data: data, CreatedAt: now, UpdatedAt: now}
	m.db.runHook(m.db.hooks, "post", "create", m.table, id, data)
	m.notify(Event{Type: EventCreate, Doc: doc})
	return doc, nil
}

// FindByID retrieves a document by its ID.
func (m *Model) FindByID(id string) (Document, error) {
	if id == "" {
		return Document{}, ErrInvalidID
	}
	m.db.runHook(m.db.hooks, "pre", "find", m.table, id, nil)

	eng, err := m.db.getEngine(m.table)
	if err != nil {
		return Document{}, err
	}
	eng.mu.RLock()
	loc, ok := eng.idx.getPrimary(id)
	if !ok {
		eng.mu.RUnlock()
		return Document{}, ErrNotFound
	}
	doc, err := eng.readDoc(id, loc)
	eng.mu.RUnlock()

	if err != nil {
		return Document{}, err
	}
	m.db.runHook(m.db.hooks, "post", "find", m.table, id, doc.Data)
	return doc, nil
}

// UpdateByID replaces the data of an existing document.
func (m *Model) UpdateByID(id string, data map[string]any) error {
	if id == "" {
		return ErrInvalidID
	}
	m.db.runHook(m.db.hooks, "pre", "update", m.table, id, data)

	eng, err := m.db.getEngine(m.table)
	if err != nil {
		return err
	}
	eng.mu.Lock()
	oldLoc, ok := eng.idx.getPrimary(id)
	if !ok {
		eng.mu.Unlock()
		return ErrNotFound
	}
	// read old doc to preserve createdAt and gather old index data
	oldDoc, err := eng.readDoc(id, oldLoc)
	if err != nil {
		eng.mu.Unlock()
		return err
	}
	now := time.Now()
	payload, err := marshalPayload(data, oldDoc.CreatedAt, now)
	if err != nil {
		eng.mu.Unlock()
		return err
	}
	r := record{
		rtype:     recUpdate,
		table:     m.table,
		id:        id,
		timestamp: now.UnixNano(),
		payload:   payload,
	}
	loc, err := eng.writeRecord(r)
	if err != nil {
		eng.mu.Unlock()
		return err
	}
	eng.idx.setPrimary(id, loc)
	eng.idx.updateSecondary(id, oldDoc.Data, data)
	eng.idx.updateText(id, oldDoc.Data, data)
	eng.mu.Unlock()

	m.db.runHook(m.db.hooks, "post", "update", m.table, id, data)
	m.notify(Event{Type: EventUpdate, Doc: Document{ID: id, Data: data, CreatedAt: oldDoc.CreatedAt, UpdatedAt: now}})
	return nil
}

// DeleteByID removes a document by its ID.
func (m *Model) DeleteByID(id string) error {
	if id == "" {
		return ErrInvalidID
	}
	m.db.runHook(m.db.hooks, "pre", "delete", m.table, id, nil)

	eng, err := m.db.getEngine(m.table)
	if err != nil {
		return err
	}
	eng.mu.Lock()
	oldLoc, ok := eng.idx.getPrimary(id)
	if !ok {
		eng.mu.Unlock()
		return ErrNotFound
	}
	oldDoc, err := eng.readDoc(id, oldLoc)
	if err != nil {
		eng.mu.Unlock()
		return err
	}
	r := record{
		rtype:     recDelete,
		table:     m.table,
		id:        id,
		timestamp: time.Now().UnixNano(),
	}
	if _, err := eng.writeRecord(r); err != nil {
		eng.mu.Unlock()
		return err
	}
	eng.idx.deletePrimary(id)
	eng.idx.removeSecondary(id)
	eng.idx.removeText(id)
	eng.mu.Unlock()

	m.db.runHook(m.db.hooks, "post", "delete", m.table, id, oldDoc.Data)
	m.notify(Event{Type: EventDelete, Doc: oldDoc})
	return nil
}

// Find returns a Query builder for this model.
func (m *Model) Find() *Query {
	return newQuery(m)
}

// execQuery is called by Query.Exec() to run the query.
func (m *Model) execQuery(q *Query) ([]Document, error) {
	eng, err := m.db.getEngine(m.table)
	if err != nil {
		return nil, err
	}

	eng.mu.RLock()

	// Try secondary index for the first equality condition on an indexed field.
	var candidateIDs []string
	candidateSet := make(map[string]bool)
	usedSecondary := false
	for _, cond := range q.conditions {
		if cond.Op == OpEq && eng.idx.hasSecondaryField(cond.Field) {
			ids := eng.idx.lookupSecondary(cond.Field, anyToString(cond.Value))
			for _, id := range ids {
				candidateSet[id] = true
			}
			candidateIDs = ids
			usedSecondary = true
			break
		}
	}
	if !usedSecondary {
		candidateIDs = eng.idx.allPrimaryIDs()
	}

	var results []Document
	skipped := 0
	for _, id := range candidateIDs {
		if q.limit > 0 && len(results) >= q.limit {
			break
		}
		loc, ok := eng.idx.getPrimary(id)
		if !ok {
			continue
		}
		doc, err := eng.readDoc(id, loc)
		if err != nil {
			continue
		}
		if !matchesConditions(doc, q.conditions) {
			continue
		}
		if skipped < q.skip {
			skipped++
			continue
		}
		results = append(results, doc)
	}
	eng.mu.RUnlock()
	return results, nil
}

// Search performs a text search and returns all matching documents.
func (m *Model) Search(text string) ([]Document, error) {
	eng, err := m.db.getEngine(m.table)
	if err != nil {
		return nil, err
	}
	eng.mu.RLock()
	ids := eng.idx.searchText(text)
	var docs []Document
	for _, id := range ids {
		loc, ok := eng.idx.getPrimary(id)
		if !ok {
			continue
		}
		doc, err := eng.readDoc(id, loc)
		if err != nil {
			continue
		}
		docs = append(docs, doc)
	}
	eng.mu.RUnlock()
	return docs, nil
}

// Watch registers a listener for document change events on this model.
// Returns an unsubscribe function.
func (m *Model) Watch(fn func(Event)) func() {
	id := atomic.AddUint64(&watcherCounter, 1)
	w := &watcher{id: id, fn: fn}
	m.mu.Lock()
	m.watchers = append(m.watchers, w)
	m.mu.Unlock()
	return func() {
		m.mu.Lock()
		for i, w2 := range m.watchers {
			if w2.id == id {
				m.watchers = append(m.watchers[:i], m.watchers[i+1:]...)
				break
			}
		}
		m.mu.Unlock()
	}
}

func (m *Model) notify(e Event) {
	m.mu.RLock()
	ws := make([]*watcher, len(m.watchers))
	copy(ws, m.watchers)
	m.mu.RUnlock()
	for _, w := range ws {
		w.fn(e)
	}
}
