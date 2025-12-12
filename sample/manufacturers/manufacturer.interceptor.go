package manufacturers

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
)

type ManufacturerInterceptor struct {
	common.Logger
}

func (instance ManufacturerInterceptor) Intercept(c ginject.Context, aggregation ginject.Aggregation) any {
	fmt.Println("[Module][Pre] Manufacturer interceptor")

	return aggregation.Pipe(
		aggregation.Consume(func(c ginject.Context, data any) any {
			fmt.Println("[Module][Post] Manufacturer interceptor")
			return data
		}),
	)
}
