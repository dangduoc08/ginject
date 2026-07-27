package benchmarks

import (
	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogModuleHTTPInterceptor struct {
	common.Logger
}

func (instance LogModuleHTTPInterceptor) Intercept(c ginject.HTTPContext, aggregation ginject.Aggregation) any {

	if c.Query().Get("error_module_http_pre_intercept") == "true" {
		panic(exception.InternalServerErrorException("LogModuleHTTPInterceptor error triggered"))
	}

	return aggregation.Pipe(
		aggregation.Transform(func(data any) any {

			if c.Query().Get("error_module_http_post_intercept") == "true" {
				panic(exception.InternalServerErrorException("LogModuleHTTPInterceptor error triggered"))
			}

			transformedData := ginject.Map{
				"data": data,
			}
			return transformedData
		}),
	)
}
