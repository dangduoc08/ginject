package list

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
)

type ListController struct {
	common.Rest
	common.Guard
	common.Interceptor
	ListProvider ListProvider
}

func (listController ListController) NewController() core.Controller {
	listController.
		Prefix("v1")

	listController.BindGuard(
		APIKeyGuard{},
		listController.CREATE_lists,
	)

	listController.BindInterceptor(
		CacheGetCompaniesInterceptor{},
		listController.READ_lists,
	)

	return listController
}

func (listController ListController) CREATE_lists(
	c gooh.Context,
	b gooh.Body,
	q gooh.Query,
	h gooh.Header,
	p gooh.Param,
	bodyDTO CreateListBody,
) any {
	return listController.ListProvider.Handler(c, p, q, h, b)
}

func (listController ListController) READ_lists(
	c gooh.Context,
	b gooh.Body,
	q gooh.Query,
	h gooh.Header,
	p gooh.Param,
	query ReadCompaniesQuery,
) any {
	return listController.ListProvider.Handler(c, p, q, h, b)
}
