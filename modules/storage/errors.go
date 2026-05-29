package storage

import "errors"

var (
	ErrNotFound     = errors.New("store: document not found")
	ErrCorrupt      = errors.New("store: corrupted record")
	ErrInvalidTable = errors.New("store: invalid table name")
	ErrInvalidID    = errors.New("store: invalid document id")
	ErrClosed       = errors.New("store: database is closed")
	ErrTxAborted    = errors.New("store: transaction aborted")
)
