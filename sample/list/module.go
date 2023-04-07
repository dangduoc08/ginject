package list

import (
	"github.com/dangduoc08/gooh/core"
)

var ListModule = core.ModuleBuilder().
	Controllers(
		ListController{},
	).
	Providers(
		ListProvider{},
	).
	Build()
