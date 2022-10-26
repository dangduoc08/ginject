package categories

import (
	"log"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/sample/products"
)

var Module = func() *common.Module {
	module := common.ModuleBuilder().
		Imports(
			products.Module,
		).
		Providers(
			CategoryProvider{},
		).
		Exports(
			CategoryProvider{},
		).
		Presenters(
			CategoryPresenter{},
		).
		Build()

	module.OnInit = func() {
		log.Default().Println("CategoriesModule OnInit")
	}

	return module
}()
