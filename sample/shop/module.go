package shop

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/accounts"
	"github.com/dangduoc08/ginject/sample/shop/catalog"
)

// Module composes the shop sample from two independent feature modules —
// accounts (identity, credentials and sessions) and catalog (per-owner store,
// category and product management) — plus UsersController, the one workflow
// that spans both: provisioning a store when an account registers.
var Module = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports(accounts.Module, catalog.Module).
		Controllers(UsersController{}).
		Build()

	return module
}
