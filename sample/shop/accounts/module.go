package accounts

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/infra"
)

// Module owns account identity end to end: registering accounts, verifying
// credentials, and bearer-token sessions backed by the cache module. It
// exports AuthGuard and CurrentUser so other feature modules can gate their
// own routes behind authentication.
var Module = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports(infra.StorageModule, infra.CacheModule).
		Providers(UserService{}).
		Controllers(SessionsController{}).
		Build()

	return module
}
