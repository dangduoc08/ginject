package three

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/five"
)

var Module = func() *core.Module {
	provider := Provider{
		Name: "3rd static provider",
	}

	controller := Controller{
		Name: "3rd static controller",
	}

	fmt.Println("3rd static module")

	mod := core.ModuleBuilder().
		Imports(five.Module).
		Providers(provider).
		Exports(provider).
		Controllers(controller).
		Build()

	return mod
}()
