package manufacturers

import "github.com/dangduoc08/ginject/core"

var ManufacturerModule = func() *core.Module {
	manufacturerController := ManufacturerController{}

	var module = core.ModuleBuilder().
		Imports().
		Controllers(manufacturerController).
		Build()

	module.
		Prefix("manufacturers")

	// module.Middleware.
	// 	Apply(
	// 		ManufacturerMiddleware1,
	// 		manufacturerController.CREATE_VERSION_1,
	// 	).
	// 	Apply(
	// 		ManufacturerMiddleware2,
	// 		manufacturerController.CREATE_VERSION_1,
	// 	)

	return module
}
