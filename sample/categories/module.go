package categories

import (
	"log"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/products"
)

var Module = func() *core.Module {
	module := core.ModuleBuilder().
		Imports(
			products.Module,
		).
		Providers(
			CategoryProvider{},
		).
		Exports(
			CategoryProvider{},
		).
		Controllers(
			CategoryController{},
		).
		Build()

	module.OnInit = func() {
		log.Default().Println("CategoriesModule OnInit")
	}

	return module
}()
