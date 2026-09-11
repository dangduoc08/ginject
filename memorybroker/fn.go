package memorybroker

import (
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/crypto"
)

func newID() string {
	id, err := crypto.UUID()
	if err != nil {
		panic(color.FmtRed("broker: failed to generate UUID: %v", err))
	}
	return id
}
