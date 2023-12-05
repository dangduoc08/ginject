package two

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/five"
	"github.com/dangduoc08/gooh/sample/three"
)

var Module = func() *core.Module {
	provider := Provider{
		Name: "2nd static provider",
	}

	controller := Controller{
		Name: "2nd static controller",
	}

	fmt.Println("2nd static module")

	mod := core.ModuleBuilder().
		Imports(three.Module, five.Module).
		Providers(provider).
		Exports(provider).
		Controllers(controller).
		Build()

	return mod
}()
