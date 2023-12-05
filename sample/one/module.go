package one

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/five"
	"github.com/dangduoc08/gooh/sample/three"
)

var Module = func() *core.Module {
	provider := Provider{
		Name: "1st static provider",
	}

	controller := Controller{
		Name: "1st static controller",
	}

	fmt.Println("1st static module")

	mod := core.ModuleBuilder().
		Imports(three.Module, five.Module).
		Providers(provider).
		Exports(provider).
		Controllers(controller).
		Build()

	return mod
}()
