package list

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type ListProvider struct {
	ConfigService config.ConfigService
	Logger        common.Logger
}

func (listProvider ListProvider) NewProvider() core.Provider {
	return listProvider
}

func (listProvider ListProvider) Handler(
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
		"envs":     listProvider.ConfigService.Get("TEST_ENV"),
	}
}
