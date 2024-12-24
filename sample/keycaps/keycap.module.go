package keycaps

import (
	"github.com/dangduoc08/gogo/core"
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
