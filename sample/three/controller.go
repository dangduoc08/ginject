package three

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
)

type Controller struct {
	common.Rest
	Name     string
	Provider Provider
}

func (controller Controller) NewController() core.Controller {
	return controller
}

func (controller Controller) READ_three() any {

	return gooh.Map{}
}
