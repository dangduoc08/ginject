package keycaps

import (
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
)

type KeycapController struct {
	common.REST
}

func (instance KeycapController) NewController() core.Controller {

	return instance
}

func (instance KeycapController) CREATE_VERSION_1() {

}

func (instance KeycapController) READ_VERSION_1() {

}
