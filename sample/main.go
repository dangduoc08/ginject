package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/middlewares/cors"
	"github.com/dangduoc08/ginject/middlewares/helmet"
	"github.com/dangduoc08/ginject/sample/benchmarks"
	"github.com/dangduoc08/ginject/sample/confs"
	"github.com/dangduoc08/ginject/sample/shared"
	"github.com/dangduoc08/ginject/sample/shop"
)

func main() {
	app := core.New()
	logger := log.NewLog(&log.LogOptions{
		Level:     log.DebugLevel,
		LogFormat: log.PrettyFormat,
	})

	app.
		UseLogger(logger).
		BindGlobalMiddlewares(cors.CORS{}, helmet.Helmet{}).
		BindGlobalInterceptors(shared.ResponseInterceptor{})

	// app.
	// 	EnableVersioning(versioning.Versioning{
	// 		Type: versioning.HeaderVersion,
	// 		Key:  confs.ENV.APIVersionName,
	// 	}).
	// 	EnableDevtool()

	app.Create(
		core.ModuleBuilder().
			Imports(benchmarks.Module, confs.ConfModule, shop.Module).
			Build(),
	)

	app.Logger.Fatal("AppError", "error", app.Listen(confs.ENV.Port))
}
