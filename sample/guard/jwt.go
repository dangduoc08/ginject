package guard

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/modules/config"
)

type JWTGuard struct {
	ConfigService config.ConfigService
}

func (jwtGuard JWTGuard) NewGuard() common.Guarder {
	fmt.Println("JWTGuard NewGuard")
	return jwtGuard
}

func (jwtGuard JWTGuard) CanActivate(c gooh.Context) bool {
	fmt.Println("JWTGuard invoke CanActivate")

	return true
}
