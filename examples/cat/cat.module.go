package cat

import "github.com/dangduoc08/gogo/core"

var CatModule = func() *core.Module {
	var module = core.ModuleBuilder().
		Providers(CatProvider{}).
		Controllers(CatController{}).
		Build()

	module.
		Prefix("pets")

	return module
}
