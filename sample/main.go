package main

import (
	"github.com/dangduoc08/gogo/core"
	"github.com/dangduoc08/gogo/log"
	"github.com/dangduoc08/gogo/middlewares"
	"github.com/dangduoc08/gogo/sample/confs"
	"github.com/dangduoc08/gogo/sample/keycaps"
	"github.com/dangduoc08/gogo/sample/manufacturers"
	"github.com/dangduoc08/gogo/sample/shared"
	"github.com/dangduoc08/gogo/versioning"
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
