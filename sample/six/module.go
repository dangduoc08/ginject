package six

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/four"
	"github.com/dangduoc08/gooh/sample/seven"
)

var Module = func(fourProvider four.Provider) *core.Module {
	provider := Provider{
		Name: "6th dynamic provider",
	}

	controller := Controller{
		Name: "6th dynamic controller",
	}

	fmt.Println("6th dynamic module see", fourProvider.Name)

	mod := core.ModuleBuilder().
		Imports(seven.Module).
		Providers(provider).
		Exports(provider).
		Controllers(controller).
		Build()

	mod.IsGlobal = true

	return mod
}
