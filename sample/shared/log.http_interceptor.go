package shared

import (
	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/exception"
)

type LogHTTPInterceptor struct {
	common.Logger
}

func (instance LogHTTPInterceptor) Intercept(c ginject.HTTPContext, aggregation ginject.Aggregation) any {

	if c.Query().Get("error_http_pre_intercept") == "true" {
		panic(exception.InternalServerErrorException("LogHTTPInterceptor error triggered"))
	}

	return aggregation.Pipe(
		aggregation.Transform(func(data any) any {

			if c.Query().Get("error_http_post_intercept") == "true" {
				panic(exception.InternalServerErrorException("LogHTTPInterceptor error triggered"))
			}

			transformedData := ginject.Map{
				"data": data,
			}
			return transformedData
		}),
	)
}
