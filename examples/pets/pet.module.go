package pets

import (
	"github.com/dangduoc08/gogo/core"
)

var Module = core.
	ModuleBuilder().
	Controllers(PetController{}).
	Build()
