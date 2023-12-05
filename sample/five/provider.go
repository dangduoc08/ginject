package five

import (
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
)

type Provider struct {
	Name   string
	Logger common.Logger
}

func (provider Provider) NewProvider() core.Provider {
	return provider
}
