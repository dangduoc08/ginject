package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/guards/throttler"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/middlewares/cors"
	"github.com/dangduoc08/ginject/middlewares/helmet"
	"github.com/dangduoc08/ginject/middlewares/requestlogger"
	"github.com/dangduoc08/ginject/sample/benchmarks"
	"github.com/dangduoc08/ginject/sample/confs"
	"github.com/dangduoc08/ginject/sample/shared"
)

func main() {
	app := core.New()
	logger := log.NewLog(&log.LogOptions{
		Level:     log.DebugLevel,
		LogFormat: log.PrettyFormat,
	})

	app.
		UseLogger(logger).
		BindGlobalMiddlewares(cors.CORS{}, requestlogger.RequestLogger{}, helmet.Helmet{}).
		BindGlobalInterceptors(shared.ResponseInterceptor{}).
		BindGlobalGuards(throttler.Throttler{})

	// app.
	// 	EnableVersioning(versioning.Versioning{
	// 		Type: versioning.HEADER,
	// 		Key:  confs.ENV.APIVersionName,
	// 	}).
	// 	EnableDevtool()

	app.Create(
		core.ModuleBuilder().
			Imports(benchmarks.Module, confs.ConfModule).
			Build(),
	)

	app.Logger.Fatal("AppError", "error", app.Listen(confs.ENV.Port))
}
