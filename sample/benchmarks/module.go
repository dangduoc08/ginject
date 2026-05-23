package benchmarks

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/modules/httpclient"
)

var Module = func() *core.Module {
	var module = core.ModuleBuilder().
		Imports(
			httpclient.Register(nil),
		).
		Controllers(Controller{}).
		Build()

	return module
}
