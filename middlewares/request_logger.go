package middlewares

import (
	"fmt"
	"time"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/context"
)

func RequestLogger(logger common.Logger) func(*context.Context) {
	return func(c *context.Context) {
		c.Event.On(context.REQUEST_FINISHED, func(args ...any) {
			newC := args[0].(*context.Context)
			responseTime := time.Now().UnixMilli() - newC.Timestamp.UnixMilli()
			logger.Info(
				newC.URL.String(),
				"Method", newC.Method,
				"Status", newC.Code,
				"Time", fmt.Sprintf("%v ms", responseTime),
				"User-Agent", newC.UserAgent(),
			)
		})

		c.Next()
	}
}
