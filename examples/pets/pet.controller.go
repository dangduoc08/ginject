package pets

import (
	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
	"github.com/dangduoc08/gogo/core"
	"github.com/dangduoc08/gogo/ctx"
)

type Guarder struct{}

func (instance Guarder) CanActivate(ctx *ctx.Context) bool {

	return true
}

type PetController struct {
	common.REST
	common.Logger
	common.Guard
}

func (instance PetController) NewController() core.Controller {
	instance.BindGuard(Guarder{})
	// instance.Prefix("v1")

	return instance
}

func (instance PetController) READ_hitpays(body gogo.Body) gogo.Map {
	instance.Info("HITPAY Webhook", "body", body)

	return gogo.Map{
		"name": "John",
	}
}
