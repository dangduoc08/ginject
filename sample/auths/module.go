package auths

import (
	"log"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/categories"
	"github.com/dangduoc08/gooh/sample/products"
)

var Module = func() *core.Module {
	module := core.ModuleBuilder().
		Imports(
			products.Module,
			categories.Module,
		).
		Providers(
			AuthProvider{},
		).
		Controllers(
			AuthController{},
		).
		Build()

	module.OnInit = func() {
		log.Default().Println("AuthsModule OnInit")
	}
	module.IsGlobal = true

	return module
}()
