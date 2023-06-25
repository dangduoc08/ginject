package book

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type BookProvider struct {
	ConfigService config.ConfigService
}

func (bookProvider BookProvider) NewProvider() core.Provider {
	return bookProvider
}

func (bookProvider BookProvider) Handler(
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
		"envs":     bookProvider.ConfigService.Config,
	}
}
