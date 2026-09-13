package storage

import (
	"path/filepath"
	"sync"
	"unicode"
)

type DB struct {
	mu             sync.RWMutex
	path           string
	enginesByTable map[string]*engine
	modelsByTable  map[string]*Model
	hooks          *hookSet
	isClosed       bool
}

func Open(path string) (*DB, error) {
	db := &DB{
		path:           path,
		enginesByTable: make(map[string]*engine),
		modelsByTable:  make(map[string]*Model),
		hooks:          newHookSet(),
	}
	return db, nil
}

func (db *DB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.isClosed {
		return ErrClosed
	}
	db.isClosed = true
	for _, eng := range db.enginesByTable {
		eng.close()
	}
	return nil
}

func (db *DB) Flush() error {
	db.mu.RLock()
	engs := make([]*engine, 0, len(db.enginesByTable))
	for _, eng := range db.enginesByTable {
		engs = append(engs, eng)
	}
	db.mu.RUnlock()
	for _, eng := range engs {
		if err := eng.flush(); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) Compact() error {
	db.mu.RLock()
	engs := make([]*engine, 0, len(db.enginesByTable))
	for _, eng := range db.enginesByTable {
		engs = append(engs, eng)
	}
	db.mu.RUnlock()
	for _, eng := range engs {
		if err := eng.compact(); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) Model(table string) *Model {
	if err := validateTableName(table); err != nil {
		panic(err)
	}
	db.mu.Lock()
	m, ok := db.modelsByTable[table]
	if !ok {
		m = newModel(db, table)
		db.modelsByTable[table] = m
	}
	db.mu.Unlock()
	return m
}

func (db *DB) Tx(fn func(*Tx) error) error {
	tx := newTx(db)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.commit()
}

func (db *DB) Use(fn func(*HookCtx, func())) *DB {

	db.hooks.mu.Lock()
	db.hooks.preByEvent["*"] = append(db.hooks.preByEvent["*"], func(hc *HookCtx) {
		fn(hc, func() {})
	})
	db.hooks.mu.Unlock()
	return db
}

func (db *DB) Pre(event string, fn func(*HookCtx)) *DB {
	db.hooks.mu.Lock()
	db.hooks.preByEvent[event] = append(db.hooks.preByEvent[event], fn)
	db.hooks.mu.Unlock()
	return db
}

func (db *DB) Post(event string, fn func(*HookCtx)) *DB {
	db.hooks.mu.Lock()
	db.hooks.postByEvent[event] = append(db.hooks.postByEvent[event], fn)
	db.hooks.mu.Unlock()
	return db
}

func (db *DB) getEngine(table string) (*engine, error) {
	db.mu.RLock()
	if db.isClosed {
		db.mu.RUnlock()
		return nil, ErrClosed
	}
	eng, ok := db.enginesByTable[table]
	db.mu.RUnlock()
	if ok {
		return eng, nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if eng, ok = db.enginesByTable[table]; ok {
		return eng, nil
	}
	dir := filepath.Join(db.path, table)
	var err error
	eng, err = openEngine(dir)
	if err != nil {
		return nil, err
	}
	db.enginesByTable[table] = eng
	return eng, nil
}

func (db *DB) runHook(hs *hookSet, phase, event, table, id string, data map[string]any) {
	hs.mu.RLock()
	var fns []hookFn
	if phase == "pre" {
		fns = append(fns, hs.preByEvent["*"]...)
		fns = append(fns, hs.preByEvent[event]...)
	} else {
		fns = append(fns, hs.postByEvent[event]...)
	}
	hs.mu.RUnlock()
	if len(fns) == 0 {
		return
	}
	hc := &HookCtx{Event: event, Table: table, ID: id, Data: data}
	for _, fn := range fns {
		fn(hc)
	}
}

func validateTableName(name string) error {
	if name == "" {
		return ErrInvalidTable
	}
	for _, ch := range name {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return ErrInvalidTable
		}
	}
	return nil
}
