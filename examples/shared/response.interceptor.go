package shared

import (
	"github.com/dangduoc08/gogo"
)

type ResponseInterceptor struct {
}

func (instance ResponseInterceptor) Intercept(c gogo.Context, aggregation gogo.Aggregation) any {
	return aggregation.Pipe(
		aggregation.Consume(func(c gogo.Context, data any) any {
			transformedData := gogo.Map{
				"data": data,
			}
			return transformedData
		}),
	)
}
