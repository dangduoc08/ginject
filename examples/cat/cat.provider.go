package cat

import (
	"github.com/dangduoc08/gogo/core"
)

type CatProvider struct{}

func (instance CatProvider) NewProvider() core.Provider {
	return instance
}
