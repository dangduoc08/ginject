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
			requestType := newC.GetType()
			responseTime := time.Now().UnixMilli() - newC.Timestamp.UnixMilli()

			if requestType == context.HTTPType {
				logger.Info(
					newC.URL.String(),
					"Method", newC.Method,
					"Status", newC.Code,
					"Time", fmt.Sprintf("%v ms", responseTime),
					"Protocol", newC.Request.Proto,
					"User-Agent", newC.UserAgent(),
					"X-Request-ID", newC.GetID(),
				)
			} else if requestType == context.WSType {
				logger.Info(
					newC.WS.Message.Event,
					"Time", fmt.Sprintf("%v ms", responseTime),
					"Subprotocol", newC.WS.GetSubprotocol(),
					"User-Agent", newC.UserAgent(),
				)
			}
		})

		c.Next()
	}
}
