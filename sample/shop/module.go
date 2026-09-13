package shop

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/accounts"
	"github.com/dangduoc08/ginject/sample/shop/catalog"
)

var Module = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports(accounts.Module, catalog.Module).
		Controllers(UsersController{}).
		Build()

	return module
}
