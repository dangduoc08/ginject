package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
)

type ResponseInterceptor struct {
	common.Logger
}

func (instance ResponseInterceptor) Intercept(c ginject.HTTPContext, aggregation ginject.Aggregation) any {
	fmt.Println("[Global] Pre interceptor")
	return aggregation.Pipe(
		aggregation.Transform(func(c ginject.HTTPContext, data any) any {
			fmt.Println("[Global] Post interceptor")
			transformedData := ginject.Map{
				"data": data,
			}
			return transformedData
		}),
	)
}
