package global

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/modules/config"
)

type PermissionGuard struct {
	ConfigService config.ConfigService
	Config        map[string]bool
}

func (g PermissionGuard) CanActivate(c gooh.Context) bool {
	return true
}
