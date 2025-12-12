package keycaps

import (
	"github.com/dangduoc08/ginject/core"
)

var KeycapModule = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports().
		Controllers(KeycapController{}).
		Build()

	module.
		Prefix("keycaps")

	return module
}
