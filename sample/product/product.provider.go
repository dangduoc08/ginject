package product

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type ProductProvider struct {
	ConfigService config.ConfigService
}

func (productProvider ProductProvider) NewProvider() core.Provider {
	return productProvider
}

func (productProvider ProductProvider) Handler(
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
		"TEST_ENV": productProvider.ConfigService.Get("TEST_ENV"),
	}
}
