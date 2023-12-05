package main

import (
	"strconv"

	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/log"
	"github.com/dangduoc08/gooh/middlewares"
	"github.com/dangduoc08/gooh/modules/config"
	"github.com/dangduoc08/gooh/sample/four"
	"github.com/dangduoc08/gooh/sample/global"
	"github.com/dangduoc08/gooh/sample/one"
	"github.com/dangduoc08/gooh/sample/seven"
	"github.com/dangduoc08/gooh/sample/six"
	"github.com/dangduoc08/gooh/sample/two"
)

func main() {
	app := core.New()
	logger := log.NewLog(&log.LogOptions{
		Level:     log.DebugLevel,
		LogFormat: log.PrettyFormat,
	})

	app.
		UseLogger(logger).
		Use(middlewares.CORS(), middlewares.RequestLogger(logger)).
		BindGlobalInterceptors(global.LoggingInterceptor{}, global.ResponseInterceptor{})

	m := core.ModuleBuilder().
		Imports(
			one.Module,
			two.Module,
			four.Module,
			six.Module,
			seven.Module,
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
		).
		Build()

	app.Create(m)

	configService := app.Get(config.ConfigService{}).(config.ConfigService)

	app.Logger.Fatal("AppError", "errMsg", app.Listen(configService.Get("PORT").(int)))
}
