package company

import (
	"github.com/dangduoc08/gooh/core"
)

var CompanyModule = func() *core.Module {
	companyController := CompanyController{}
	companyProvider := CompanyProvider{}

	module := core.ModuleBuilder().
		Controllers(
			companyController,
		).
		Providers(
			companyProvider,
		).
		Exports(
			companyProvider,
		).
		Build()

	// module.Middleware.
	// 	Apply(
	// 		func(c gooh.Context) {
	// 			fmt.Println("CompanyModule Middleware1")
	// 			c.Next()
	// 		},
	// 		func(c gooh.Context) {
	// 			fmt.Println("CompanyModule Middleware2")
	// 			c.Next()
	// 		},
	// 	).
	// 	Exclude([]any{
	// 		companyController.READ,
	// 	})

	return module
}()
