package cat

import (
	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
)

type CatGuard struct {
	common.Logger
}

func (instance CatGuard) NewGuard() CatGuard {
	return instance
}

func (instance CatGuard) CanActivate(c gogo.Context) bool {
	return true
}
