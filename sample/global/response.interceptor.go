package global

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/modules/config"
)

type ResponseInterceptor struct {
	ConfigService config.ConfigService
}

func (i ResponseInterceptor) Intercept(c gooh.Context, aggregation gooh.Aggregation) any {
	return aggregation.Pipe(
		aggregation.Consume(func(data any) any {
			transformedData := gooh.Map{
				"data": data,
			}
			return transformedData
		}),
	)
}
