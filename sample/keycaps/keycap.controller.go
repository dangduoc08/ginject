package keycaps

import (
	"github.com/dangduoc08/gogo/common"
	"github.com/dangduoc08/gogo/core"
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
