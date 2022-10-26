package auths

import (
	"log"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/sample/categories"
	"github.com/dangduoc08/gooh/sample/products"
)

var Module = func() *common.Module {
	module := common.ModuleBuilder().
		Imports(
			products.Module,
			categories.Module,
		).
		Providers(
			AuthProvider{},
		).
		Presenters(
			AuthPresenter{},
		).
		Build()

	module.OnInit = func() {
		log.Default().Println("AuthsModule OnInit")
	}

	return module
}()
