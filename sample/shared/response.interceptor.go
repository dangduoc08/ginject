package shared

import (
	"fmt"

	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
)

type ResponseInterceptor struct {
	common.Logger
}

func (instance ResponseInterceptor) Intercept(c gogo.Context, aggregation gogo.Aggregation) any {
	fmt.Println("[Global][Pre] Response interceptor")

	return aggregation.Pipe(
		aggregation.Consume(func(c gogo.Context, data any) any {
			fmt.Println("[Global][Post] Response interceptor")
			transformedData := gogo.Map{
				"data": data,
			}
			return transformedData
		}),
	)
}
