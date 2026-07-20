package requestlogger

import (
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

type RequestLogger struct {
	common.Logger
}

func (instance RequestLogger) Use(c *ctx.HTTPContext, next ctx.Next) {
	c.Event.Once(ctx.RequestFinished, func(args ...any) {
		newC := args[0].(*ctx.HTTPContext)

		var msg string = newC.URL.String()

		instance.Info(
			msg,
			"Method", newC.Method,
			"Status", newC.Code,
			"Time", time.Since(newC.Timestamp).String(),
			"Protocol", newC.Proto,
			"User-Agent", newC.UserAgent(),
			ctx.RequestID, newC.GetID(),
		)
	})

	next()
}
