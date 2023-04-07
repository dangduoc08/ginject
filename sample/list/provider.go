package list

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type ListProvider struct {
	ConfigService config.ConfigService
}

func (listProvider ListProvider) handler(
	c gooh.Context,
	p gooh.Param,
	q gooh.Query,
) any {
	return gooh.Map{
		"method":   c.Method,
		"route":    c.GetRoute(),
		"original": c.URL.Path,
		"params":   p,
		"queries":  q,
		"TEST_ENV": listProvider.ConfigService.Get("TEST_ENV"),
	}
}

func (listProvider ListProvider) Inject() core.Provider {

	return listProvider
}
