package company

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type CompanyProvider struct {
	ConfigService config.ConfigService
}

func (companyProvider CompanyProvider) NewProvider() core.Provider {
	return companyProvider
}

func (companyProvider CompanyProvider) Handler(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
	b gooh.Body,
) any {
	return gooh.Map{
		"method":   c.Method,
		"route":    c.GetRoute(),
		"original": c.URL.Path,
		"params":   p,
		"queries":  q,
		"headers":  h,
		"body":     b,
		"envs":     companyProvider.ConfigService.Get("TEST_ENV"),
	}
}
