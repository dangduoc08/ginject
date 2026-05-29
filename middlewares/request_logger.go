package middlewares

import (
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

type RequestLogger struct {
	common.Logger
}

func (instance RequestLogger) Use(c *ctx.Context, next ctx.Next) {
	c.Event.On(ctx.REQUEST_FINISHED, func(args ...any) {
		newC := args[0].(*ctx.Context)
		requestType := newC.GetType()
		responseTime := time.Since(newC.Timestamp)

		switch requestType {
		case ctx.HTTPType:
			instance.Info(
				newC.URL.String(),
				"Method", newC.Method,
				"Status", newC.Code,
				"Time", responseTime.String(),
				"Protocol", newC.Proto,
				"User-Agent", newC.UserAgent(),
				ctx.REQUEST_ID, newC.GetID(),
			)
		case ctx.WSType:
			instance.Info(
				newC.WS.Message.Event,
				"Time", responseTime.String(),
				"Subprotocol", newC.WS.GetSubprotocol(),
				"User-Agent", newC.UserAgent(),
			)
		}
	})

	next()
}
