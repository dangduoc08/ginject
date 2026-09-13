package storage

import (
	"sync/atomic"
	"time"
)

var txCounter uint64

func nextTxID() uint64 {
	return atomic.AddUint64(&txCounter, 1)
}

type txOp struct {
	r record
}

type Tx struct {
	db          *DB
	id          uint64
	ops         []txOp
	docsByTable map[string]map[string]Document
}

func newTx(db *DB) *Tx {
	return &Tx{
		db:          db,
		id:          nextTxID(),
		docsByTable: make(map[string]map[string]Document),
	}
}

type TxModel struct {
	tx    *Tx
	table string
}

func (tx *Tx) Model(table string) *TxModel {
	return &TxModel{tx: tx, table: table}
}

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
	if tm.tx.docsByTable[tm.table] == nil {
		tm.tx.docsByTable[tm.table] = make(map[string]Document)
	}
	tm.tx.docsByTable[tm.table][id] = doc
	return doc, nil
}

func (tm *TxModel) UpdateByID(id string, data map[string]any) error {
	if id == "" {
		return ErrInvalidID
	}
	now := time.Now()

	createdAt := now
	if docs, ok := tm.tx.docsByTable[tm.table]; ok {
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

func (tx *Tx) commit() error {
	if len(tx.ops) == 0 {
		return nil
	}

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

		begin := record{rtype: recTxBegin, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
		if _, err := eng.writeRecord(begin); err != nil {
			eng.mu.Unlock()
			return err
		}

		type writtenOp struct {
			r   record
			loc location
		}
		var written []writtenOp
		for _, r := range recs {
			loc, err := eng.writeRecord(r)
			if err != nil {

				rb := record{rtype: recTxRollback, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
				_, _ = eng.writeRecord(rb)
				eng.mu.Unlock()
				return err
			}
			written = append(written, writtenOp{r: r, loc: loc})
		}

		commit := record{rtype: recTxCommit, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
		if _, err := eng.writeRecord(commit); err != nil {
			rb := record{rtype: recTxRollback, txID: tx.id, table: table, timestamp: time.Now().UnixNano()}
			_, _ = eng.writeRecord(rb)
			eng.mu.Unlock()
			return err
		}

		_ = eng.current.f.Sync()

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
