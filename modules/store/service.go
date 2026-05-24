package store

import "github.com/dangduoc08/ginject/core"

// StoreService is the injectable DI provider that wraps a *DB.
type StoreService struct {
	DB *DB
}

func (ss StoreService) NewProvider() core.Provider {
	return ss
}

// Model delegates to the underlying DB.
func (ss *StoreService) Model(table string) *Model {
	return ss.DB.Model(table)
}

// Tx delegates to the underlying DB.
func (ss *StoreService) Tx(fn func(*Tx) error) error {
	return ss.DB.Tx(fn)
}

// Flush delegates to the underlying DB.
func (ss *StoreService) Flush() error {
	return ss.DB.Flush()
}

// Compact delegates to the underlying DB.
func (ss *StoreService) Compact() error {
	return ss.DB.Compact()
}
