package products

import (
	"log"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

var Module = func() *core.Module {
	productProvider := ProductProvider{}

	module := core.ModuleBuilder().
		Imports(config.Module).
		Providers(
			productProvider,
		).
		Exports(
			productProvider,
		).
		Controllers(
			ProductController{},
		).
		Build()

	module.OnInit = func() {
		log.Default().Println("ProductModule OnInit")
	}

	return module
}()
