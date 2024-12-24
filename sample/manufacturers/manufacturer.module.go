package manufacturers

import "github.com/dangduoc08/gogo/core"

var ManufacturerModule = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports().
		Controllers(ManufacturerController{}).
		Build()

	module.
		Prefix("manufacturers")

	module.Middleware.Apply(ManufacturerMiddleware)

	return module
}
