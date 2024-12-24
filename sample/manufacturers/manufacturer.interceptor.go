package manufacturers

import (
	"fmt"

	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
)

type ManufacturerInterceptor struct {
	common.Logger
}

func (instance ManufacturerInterceptor) Intercept(c gogo.Context, aggregation gogo.Aggregation) any {
	fmt.Println("[Module][Pre] Manufacturer interceptor")

	return aggregation.Pipe(
		aggregation.Consume(func(c gogo.Context, data any) any {
			fmt.Println("[Module][Post] Manufacturer interceptor")
			return data
		}),
	)
}
