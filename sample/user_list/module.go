package userlist

import (
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

var UserListModule = func(configService config.ConfigService) *core.Module {
	return core.ModuleBuilder().
		Controllers(
			UserListController{},
		).
		Providers(
			UserListProvider{},
		).
		Build()
}
