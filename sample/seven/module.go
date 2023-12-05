package seven

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/five"
)

var Module = func() *core.Module {
	provider := Provider{
		Name: "7th static provider",
	}

	controller := Controller{
		Name: "7th static controller",
	}

	fmt.Println("7th static module")

	mod := core.ModuleBuilder().
		Imports(five.Module).
		Providers(provider).
		Exports(provider).
		Controllers(controller).
		Build()

	mod.IsGlobal = true

	return mod
}()
