package cat

import (
	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
	"github.com/dangduoc08/gogo/core"
)

type CatController struct {
	common.REST
	common.Guard
	common.Interceptor
	common.ExceptionFilter
}

func (instance CatController) NewController() core.Controller {
	instance.BindGuard(
		CatGuard{},
		instance.CREATE_VERSION_1,
	)

	instance.BindInterceptor(
		CatInterceptor{},
		instance.CREATE_VERSION_1,
	)

	instance.BindExceptionFilter(
		CatExceptionFilter{},
		instance.CREATE_VERSION_1,
	)

	instance.
		Prefix("/cats")

	return instance
}

func (instance CatController) CREATE_VERSION_1() any {
	return gogo.Map{
		"cat": "Tom",
	}
}
