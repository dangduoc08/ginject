package store

import (
	"sync/atomic"
	"time"
)

var txCounter uint64

func nextTxID() uint64 {
	return atomic.AddUint64(&txCounter, 1)
}

// txOp is a buffered operation within a transaction.
type txOp struct {
	r record
}

// Tx is an in-progress transaction. Obtain one via DB.Tx().
// All operations are buffered in memory and written atomically to disk on commit.
type Tx struct {
	db   *DB
	id   uint64
	ops  []txOp
	docs map[string]map[string]Document // table → id → doc (for within-tx reads)
}

func newTx(db *DB) *Tx {
	return &Tx{
		db:   db,
		id:   nextTxID(),
		docs: make(map[string]map[string]Document),
	}
}

// TxModel is the transaction-scoped model for a specific table.
type TxModel struct {
	tx    *Tx
	table string
}

// Model returns a transaction-scoped model for the given table.
func (tx *Tx) Model(table string) *TxModel {
	return &TxModel{tx: tx, table: table}
}

// Create buffers an insert within the transaction.
func (tm *TxModel) Create(data map[string]any) (Document, error) {
	id, err := newID()
	if err != nil {
		return Document{}, err
	}
	now := time.Now()
	payload, err := marshalPayload(data, now, now)
	if err != nil {
		return Document{}, err
	}
	doc := Document{ID: id, Data: data, CreatedAt: now, UpdatedAt: now}
	tm.tx.ops = append(tm.tx.ops, txOp{r: record{
		rtype:     recInsert,
		txID:      tm.tx.id,
		table:     tm.table,
		id:        id,
		timestamp: now.UnixNano(),
		payload:   payload,
	}})
	if tm.tx.docs[tm.table] == nil {
		tm.tx.docs[tm.table] = make(map[string]Document)
	}
	tm.tx.docs[tm.table][id] = doc
	return doc, nil
}

// UpdateByID buffers an update within the transaction.
func (tm *TxModel) UpdateByID(id string, data map[string]any) error {
	if id == "" {
		return ErrInvalidID
	}
	now := time.Now()
	// try to find createdAt from buffered ops, else use now
	createdAt := now
	if docs, ok := tm.tx.docs[tm.table]; ok {
		if d, ok := docs[id]; ok {
			createdAt = d.CreatedAt
		}
	}
	payload, err := marshalPayload(data, createdAt, now)
	if err != nil {
		return err
	}
	tm.tx.ops = append(tm.tx.ops, txOp{r: record{
		rtype:     recUpdate,
		txID:      tm.tx.id,
		table:     tm.table,
		id:        id,
		timestamp: now.UnixNano(),
		payload:   payload,
	}})
	return nil
}

// DeleteByID buffers a delete within the transaction.
func (tm *TxModel) DeleteByID(id string) error {
	if id == "" {
		return ErrInvalidID
	}
	tm.tx.ops = append(tm.tx.ops, txOp{r: record{
		rtype:     recDelete,
		txID:      tm.tx.id,
		table:     tm.table,
		id:        id,
		timestamp: time.Now().UnixNano(),
	}})
	return nil
}

// commit writes all buffered ops atomically to disk.
func (tx *Tx) commit() error {
	if len(tx.ops) == 0 {
		return nil
	}

	// group ops by table
	byTable := make(map[string][]record)
	for _, op := range tx.ops {
		byTable[op.r.table] = append(byTable[op.r.table], op.r)
	}

	for table, recs := range byTable {
		eng, err := tx.db.getEngine(table)
		if err != nil {
			return err
		}

		eng.mu.Lock()

		// write TX_BEGIN
		begin := record{rtype: recTxBegin, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
		if _, err := eng.writeRecord(begin); err != nil {
			eng.mu.Unlock()
			return err
		}

		// write all ops
		type writtenOp struct {
			r   record
			loc location
		}
		var written []writtenOp
		for _, r := range recs {
			loc, err := eng.writeRecord(r)
			if err != nil {
				// write rollback marker and abort
				rb := record{rtype: recTxRollback, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
				_, _ = eng.writeRecord(rb)
				eng.mu.Unlock()
				return err
			}
			written = append(written, writtenOp{r: r, loc: loc})
		}

		// write TX_COMMIT
		commit := record{rtype: recTxCommit, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
		if _, err := eng.writeRecord(commit); err != nil {
			rb := record{rtype: recTxRollback, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
			_, _ = eng.writeRecord(rb)
			eng.mu.Unlock()
			return err
		}

		// sync to disk
		_ = eng.current.f.Sync()

		// apply to indexes
		for _, op := range written {
			switch op.r.rtype {
			case recInsert, recUpdate:
				op.r.txID = 0
				var oldData map[string]any
				if old, ok := eng.idx.getPrimary(op.r.id); ok {
					if seg := eng.segByID(old.segID); seg != nil {
						if prev, err := readRecordAt(seg, old); err == nil {
							oldData, _, _, _ = unmarshalPayload(prev.payload)
						}
					}
				}
				eng.idx.setPrimary(op.r.id, op.loc)
				newData, _, _, _ := unmarshalPayload(op.r.payload)
				eng.idx.updateSecondary(op.r.id, oldData, newData)
				eng.idx.updateText(op.r.id, oldData, newData)
			case recDelete:
				var oldData map[string]any
				if old, ok := eng.idx.getPrimary(op.r.id); ok {
					if seg := eng.segByID(old.segID); seg != nil {
						if prev, err := readRecordAt(seg, old); err == nil {
							oldData, _, _, _ = unmarshalPayload(prev.payload)
						}
					}
				}
				eng.idx.deletePrimary(op.r.id)
				eng.idx.removeSecondary(op.r.id)
				eng.idx.removeText(op.r.id)
				_ = oldData
			}
		}

		eng.mu.Unlock()
	}
	return nil
}
