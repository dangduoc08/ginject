package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/middlewares"
	"github.com/dangduoc08/ginject/sample/confs"
	"github.com/dangduoc08/ginject/sample/keycaps"
	"github.com/dangduoc08/ginject/sample/manufacturers"
	"github.com/dangduoc08/ginject/sample/shared"
	"github.com/dangduoc08/ginject/versioning"
)

func main() {
	app := core.New()
	logger := log.NewLog(&log.LogOptions{
		Level:     log.DebugLevel,
		LogFormat: log.PrettyFormat,
	})

	app.
		UseLogger(logger).
		BindGlobalMiddlewares(middlewares.CORS{}).
		BindGlobalInterceptors(shared.ResponseInterceptor{}).
		BindGlobalGuards(shared.RateLimiterGuard{})

	app.
		EnableVersioning(versioning.Versioning{
			Type: versioning.HEADER,
			Key:  confs.ENV.APIVersionName,
		}).
		EnableDevtool()

	app.Create(
		core.ModuleBuilder().
			Imports(keycaps.KeycapModule, manufacturers.ManufacturerModule, confs.ConfModule).
			Build().
			Prefix("apis"),
	)

	app.Logger.Fatal("AppError", "error", app.Listen(confs.ENV.Port))
}
