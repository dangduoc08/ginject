package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/modules/config"
	"github.com/dangduoc08/gooh/sample/list"
	userList "github.com/dangduoc08/gooh/sample/user_list"
)

func main() {
	app := core.New()
	app.Use(middlewares.Recovery, middlewares.RequestLogger)

	app.Create(
		core.ModuleBuilder().
			Imports(
				config.Register(config.ConfigModuleOptions{
					IsGlobal:          true,
					IsExpandVariables: true,
				}),
				userList.UserListModule,
				list.ListModule,
			).
			Build(),
	)

	configService, ok := app.Get(config.ConfigService{}).(config.ConfigService)
	if !ok {
		panic(errors.New("cannot get config.ConfigService"))
	}

	port := fmt.Sprintf(":%v", configService.Get("PORT"))

	log.Fatal(app.ListenAndServe(port))
}
