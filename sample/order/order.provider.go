package order

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type OrderProvider struct {
	ConfigService config.ConfigService
}

func (orderProvider OrderProvider) NewProvider() core.Provider {
	return orderProvider
}

func (orderProvider OrderProvider) Handler(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
	h gooh.Header,
) any {
	return gooh.Map{
		"method":   c.Method,
		"route":    c.GetRoute(),
		"original": c.URL.Path,
		"params":   p,
		"queries":  q,
		"headers":  h,
		"TEST_ENV": orderProvider.ConfigService.Get("TEST_ENV"),
	}
}
