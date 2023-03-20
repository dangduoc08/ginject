package main

import (
	"fmt"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/modules/config"
)

type DBProvider struct {
	ConfigService config.ConfigService
}

func (dbProvider DBProvider) Inject() core.Provider {
	fmt.Println("URI:", dbProvider.ConfigService.Get(("URI")))
	fmt.Println("PASSWORD:", dbProvider.ConfigService.Get(("PASSWORD")))
	return dbProvider
}

func main() {
	app := core.New()

	app.Create(
		core.ModuleBuilder().
			Imports(
				config.Register(config.ConfigModuleOptions{
					IsExpandVariables: true,
				}),
			).
			Providers(DBProvider{}).
			Build(),
	)
}
