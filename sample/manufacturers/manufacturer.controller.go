package manufacturers

import (
	"fmt"

	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
	"github.com/dangduoc08/gogo/core"
	"github.com/dangduoc08/gogo/sample/manufacturers/dtos"
	"github.com/dangduoc08/gogo/sample/shared"
)

type ManufacturerController struct {
	common.ExceptionFilter
	common.Middleware
	common.Interceptor
	common.Guard
	common.REST
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

func (instance ManufacturerController) CREATE_VERSION_1(bodyDTO dtos.CREATE_VERSION_1_Body_DTO) gogo.Map {
	fmt.Println("[Module] CREATE_VERSION_1 controller")
	return gogo.Map{
		"List": "ada",
	}
}

func (instance ManufacturerController) READ_VERSION_1(queryDTO dtos.READ_VERSION_1_Query_DTO) gogo.Map {
	fmt.Println("[Module] READ_VERSION_1 controller")
	return gogo.Map{
		"List": "ada",
	}
}

func (instance ManufacturerController) READ_BY_id_VERSION_1(queryDTO dtos.READ_BY_id_VERSION_1_Query_DTO) gogo.Map {
	fmt.Println("[Module] READ_BY_id_VERSION_1 controller")
	return gogo.Map{
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
