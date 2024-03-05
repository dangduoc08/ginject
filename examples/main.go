package main

import (
	"github.com/dangduoc08/gogo/core"
	"github.com/dangduoc08/gogo/examples/pets"
	"github.com/dangduoc08/gogo/examples/shared"
	"github.com/dangduoc08/gogo/log"
	"github.com/dangduoc08/gogo/middlewares"
	"github.com/dangduoc08/gogo/modules/config"
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
		BindGlobalInterceptors(shared.LoggingInterceptor{}, shared.ResponseInterceptor{})

	app.Create(
		core.ModuleBuilder().
			Imports(
				pets.Module,
				config.Register(&config.ConfigModuleOptions{
					IsGlobal:          true,
					IsExpandVariables: true,
					Hooks: []config.ConfigHookFn{
						func(c config.ConfigService) {
							c.Set("PORT", 3001)
						}},
				}),
			).
			Build(),
	)

	configService := app.Get(config.ConfigService{}).(config.ConfigService)

	app.Logger.Fatal("AppError", "error", app.Listen(configService.Get("PORT").(int)))
}
