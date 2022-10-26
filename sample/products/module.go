package products

import (
	"log"

	"github.com/dangduoc08/gooh/common"
)

var Module = func() *common.Module {
	module := common.ModuleBuilder().
		Providers(
			ProductProvider{},
		).
		Exports(
			ProductProvider{},
		).
		Presenters(
			ProductPresenter{},
		).
		Build()

	module.OnInit = func() {
		log.Default().Println("ProductModule OnInit")
	}

	return module
}()
