package catalog

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/accounts"
	"github.com/dangduoc08/ginject/sample/shop/infra"
)

// Module owns the per-owner catalog: one store per owner, holding its own
// categories and products. Every route is gated behind an authenticated
// owner, so it imports the accounts module for AuthGuard and CurrentUser.
var Module = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports(infra.StorageModule, accounts.Module).
		Providers(StoreService{}).
		Controllers(StoreController{}).
		Build()

	return module
}
