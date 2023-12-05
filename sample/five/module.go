package five

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/four"
)

var Module = func(fourProvider four.Provider) *core.Module {
	provider := Provider{
		Name: "5th dynamic provider",
	}

	controller := Controller{
		Name: "5th dynamic controller",
	}

	fmt.Println("5th dynamic module see", fourProvider.Name)

	mod := core.ModuleBuilder().
		Providers(provider).
		Exports(provider).
		Controllers(controller).
		Build()

	return mod
}
