package four

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
)

var Module = func() *core.Module {
	provider := Provider{
		Name: "4th static provider",
	}

	controller := Controller{
		Name: "4th static controller",
	}

	fmt.Println("4th static module")

	mod := core.ModuleBuilder().
		Providers(provider).
		Exports(provider).
		Controllers(controller).
		Build()

	mod.IsGlobal = true

	return mod
}()
