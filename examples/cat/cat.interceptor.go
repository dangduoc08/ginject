package cat

import (
	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
)

type CatInterceptor struct {
	common.Logger
}

func (instance CatInterceptor) Intercept(c gogo.Context, aggregation gogo.Aggregation) any {
	return aggregation.Pipe(
		aggregation.Consume(func(c gogo.Context, data any) any {
			return data
		}),
	)
}
