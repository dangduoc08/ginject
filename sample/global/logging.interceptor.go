package global

import (
	"encoding/json"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/modules/config"
)

type LoggingInterceptor struct {
	ConfigService config.ConfigService
	Logger        common.Logger
}

func (i LoggingInterceptor) Intercept(c gooh.Context, aggregation gooh.Aggregation) any {
	reqJSON, _ := json.Marshal(c.Body())
	i.Logger.Info("Request", string(reqJSON))
	return aggregation.Pipe(
		aggregation.Consume(func(data any) any {
			resJSON, _ := json.Marshal(data)
			i.Logger.Info("Response", string(resJSON))
			return data
		}),
	)
}
