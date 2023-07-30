package company

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/global"
)

type CompanyController struct {
	common.Rest
	common.Guard
	common.Interceptor
	common.ExceptionFilter
	CompanyProvider CompanyProvider
}

func (companyController CompanyController) NewController() core.Controller {
	companyController.
		Prefix("v1").
		Prefix("companies")

	companyController.BindExceptionFilter(global.AllExceptionsFilter{})

	companyController.BindGuard(
		APIKeyGuard{},
		companyController.CREATE,
	)

	companyController.BindInterceptor(
		CacheGetCompaniesInterceptor{},
		companyController.READ,
	)

	return companyController
}

func (companyController CompanyController) CREATE(
	c gooh.Context,
	b gooh.Body,
	q gooh.Query,
	h gooh.Header,
	p gooh.Param,
	createCompanyBody CreateCompanyBody,
) any {
	return companyController.CompanyProvider.Handler(c, p, q, h, b)
}

func (companyController CompanyController) CREATE_BY_user_id(
	c gooh.Context,
	b gooh.Body,
	q gooh.Query,
	h gooh.Header,
	p gooh.Param,
	createCompanyBody CreateCompanyBody,
) any {
	return companyController.CompanyProvider.Handler(c, p, q, h, b)
}

func (companyController CompanyController) READ(
	c gooh.Context,
	b gooh.Body,
	q gooh.Query,
	h gooh.Header,
	p gooh.Param,
	body CreateCompanyBody,
) any {
	return companyController.CompanyProvider.Handler(c, p, q, h, b)
}
