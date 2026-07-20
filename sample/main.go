package main

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/middlewares/cors"
	"github.com/dangduoc08/ginject/middlewares/helmet"
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
		BindGlobalMiddlewares(cors.CORS{}, helmet.Helmet{}, shared.LogMiddleware{}).
		BindGlobalGuards(shared.LogHTTPGuard{}, shared.LogWSGuard{}).
		BindGlobalInterceptors(shared.LogHTTPInterceptor{}, shared.LogWSInterceptor{}).
		BindGlobalExceptionFilters(shared.LogHTTPExceptionFilter{}, shared.LogWSExceptionFilter{})

	app.EnableWS(&core.WSConfig{
		Path: "ws",
	})

	// app.
	// 	EnableVersioning(versioning.Versioning{
	// 		Type: versioning.HeaderVersion,
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
