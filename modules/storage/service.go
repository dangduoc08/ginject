package storage

import "github.com/dangduoc08/ginject/core"

type StoreService struct {
	DB *DB
}

func (ss StoreService) NewProvider() core.Provider {
	return ss
}

func (ss *StoreService) Model(table string) *Model {
	return ss.DB.Model(table)
}

func (ss *StoreService) Tx(fn func(*Tx) error) error {
	return ss.DB.Tx(fn)
}

func (ss *StoreService) Flush() error {
	return ss.DB.Flush()
}

func (ss *StoreService) Compact() error {
	return ss.DB.Compact()
}
