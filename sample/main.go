package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/modules/config"
	"github.com/dangduoc08/gooh/sample/company"
	"github.com/dangduoc08/gooh/sample/global"
)

func main() {
	app := core.New()
	app.
		Use(middlewares.RequestLogger).
		BindGlobalGuards(global.PermissionGuard{}).
		BindGlobalInterceptors(global.LoggingInterceptor{}, global.ResponseInterceptor{}).
		BindGlobalExceptionFilters(global.AllExceptionsFilter{})

	app.Create(
		core.ModuleBuilder().
			Imports(
				config.Register(&config.ConfigModuleOptions{
					IsGlobal:          true,
					IsExpandVariables: true,
				}),
				company.CompanyModule,
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
