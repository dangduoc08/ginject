package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
)

type ResponseInterceptor struct {
	common.Logger
}

func (instance ResponseInterceptor) Intercept(c ginject.Context, aggregation ginject.Aggregation) any {
	fmt.Println("[Global][Pre] Response interceptor")

	return aggregation.Pipe(
		aggregation.Consume(func(c ginject.Context, data any) any {
			fmt.Println("[Global][Post] Response interceptor")
			transformedData := ginject.Map{
				"data": data,
			}
			return transformedData
		}),
	)
}
