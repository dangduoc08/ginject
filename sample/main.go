package main

import (
	"strconv"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/modules/config"
	"github.com/dangduoc08/gooh/sample/global"
	"github.com/dangduoc08/gooh/sample/list"
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
					Hooks: []config.ConfigHookFn{
						func(c config.ConfigService) {
							port := c.Get("PORT")
							if s, ok := port.(string); ok {
								port, err := strconv.Atoi(s)
								if err != nil {
									panic(err)
								}
								c.Set("PORT", port)
							}
						},
					},
				}),
				list.ListModule,
			).
			Build(),
	)

	configService := app.Get(config.ConfigService{}).(config.ConfigService)

	app.Logger.Fatal("AppError", app.Listen(configService.Get("PORT").(int)))
}
