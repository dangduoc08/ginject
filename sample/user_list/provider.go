package userlist

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type UserListProvider struct {
	ConfigService config.ConfigService
}

func (userListProvider UserListProvider) handler(
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
		"TEST_ENV": userListProvider.ConfigService.Get("TEST_ENV"),
	}
}

func (userListProvider UserListProvider) Inject() core.Provider {

	return userListProvider
}
