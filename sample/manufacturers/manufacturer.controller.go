package manufacturers

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/manufacturers/dtos"
	"github.com/dangduoc08/ginject/sample/shared"
)

type ManufacturerController struct {
	common.ExceptionFilter
	common.Middleware
	common.Interceptor
	common.Guard
	common.REST
	common.WS
}

func (instance ManufacturerController) NewController() core.Controller {
	instance.BindExceptionFilter(
		ManufacturerExceptionFilter{},
	)

	instance.BindInterceptor(
		ManufacturerInterceptor{},
	)

	instance.BindGuard(
		shared.AuthenticationGuard{},
		instance.CREATE_VERSION_1,
		instance.UPDATE_VERSION_1,
		instance.DELETE_VERSION_1,
	)

	instance.BindMiddleware(
		ManufacturerMiddleware{},
		instance.UPDATE_VERSION_1,
	)

	return instance
}

func (instance ManufacturerController) SUBSCRIBE_test() string {

	return "hihi"
}

func (instance ManufacturerController) CREATE_VERSION_1(bodyDTO dtos.CREATE_VERSION_1_Body_DTO) ginject.Map {
	fmt.Println("[Module] CREATE_VERSION_1 controller")
	return ginject.Map{
		"List": "ada",
	}
}

func (instance ManufacturerController) READ_VERSION_1(queryDTO dtos.READ_VERSION_1_Query_DTO) ginject.Map {
	fmt.Println("[Module] READ_VERSION_1 controller")
	return ginject.Map{
		"List": "ada",
	}
}

func (instance ManufacturerController) READ_BY_id_VERSION_1(queryDTO dtos.READ_BY_id_VERSION_1_Query_DTO) ginject.Map {
	fmt.Println("[Module] READ_BY_id_VERSION_1 controller")
	return ginject.Map{
		"List": "ada",
	}
}

func (instance ManufacturerController) UPDATE_VERSION_1() {
	fmt.Println("[Module] UPDATE_VERSION_1 controller")
}

func (instance ManufacturerController) MODIFY_VERSION_1() {
	fmt.Println("[Module] MODIFY_VERSION_1 controller")
}

func (instance ManufacturerController) DELETE_VERSION_1() {
	fmt.Println("[Module] DELETE_VERSION_1 controller")
}
