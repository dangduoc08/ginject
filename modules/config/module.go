package config

import (
	"github.com/dangduoc08/gooh/core"
)

var Module = func() *core.Module {
	config := map[string]string{
		"username": "root",
		"pwd":      "password",
	}

	configProvider := ConfigProvider{config}

	module := core.ModuleBuilder().
		Imports().
		Providers(
			configProvider,
		).
		Exports(
			configProvider,
		).
		Build()

	module.IsGlobal = true

	return module
}()
