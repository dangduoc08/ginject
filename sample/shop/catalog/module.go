package catalog

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/accounts"
	"github.com/dangduoc08/ginject/sample/shop/infra"
)

var Module = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports(infra.StorageModule, accounts.Module).
		Providers(StoreService{}).
		Controllers(StoreController{}).
		Build()

	return module
}
