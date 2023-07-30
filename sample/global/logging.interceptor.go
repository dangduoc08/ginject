package global

import (
	"encoding/json"
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/modules/config"
)

type LoggingInterceptor struct {
	ConfigService config.ConfigService
}

func (i LoggingInterceptor) Intercept(c gooh.Context, aggregation gooh.Aggregation) any {
	reqJSON, _ := json.Marshal(c.Body())
	fmt.Println("Request", string(reqJSON))
	return aggregation.Pipe(
		aggregation.Consume(func(data any) any {
			resJSON, _ := json.Marshal(data)
			fmt.Println("Response", string(resJSON))
			return data
		}),
	)
}
