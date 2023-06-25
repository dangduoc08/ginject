package global

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/modules/config"
)

type Guard struct {
	ConfigService config.ConfigService
}

func (guard Guard) NewGuard() common.Guarder {
	fmt.Println("Global Guard NewGuard")
	return guard
}

func (guard Guard) CanActivate(c gooh.Context) bool {
	fmt.Println("Global Guard invoke CanActivate")

	return true
}
