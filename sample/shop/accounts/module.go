package accounts

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/infra"
)

var Module = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports(infra.StorageModule, infra.CacheModule).
		Providers(UserService{}).
		Controllers(SessionsController{}).
		Build()

	return module
}
