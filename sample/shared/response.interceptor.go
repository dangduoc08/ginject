package shared

import (
	"fmt"
	"time"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
)

type ResponseInterceptor struct {
	common.Logger
}

func (instance ResponseInterceptor) Intercept(c ginject.Context, aggregation ginject.Aggregation) any {
	fmt.Println("[Global][Pre] Response interceptor")

	// Timeout is registered directly on the aggregation (not passed to Pipe)
	// so it runs in the operator pipeline regardless of Pipe's arguments.
	aggregation.Timeout(time.Second)

	return aggregation.Pipe(
		aggregation.Transform(func(c ginject.Context, data any) any {
			fmt.Println("[Post] Transform aggegator be called", data)
			transformedData := ginject.Map{
				"data": data,
			}
			return transformedData
		}),
		aggregation.Tap(func(c ginject.Context, data any) any {
			fmt.Println("[Post] Tap aggegator be called", data)
			data = ginject.Map{
				"hihi": "haha",
			}

			return data
		}),
		aggregation.Error(func(c ginject.Context, e any) any {
			fmt.Println("[Post] Error aggegator be called", e)
			c.JSON(e)

			return nil
		}),
	)
}
