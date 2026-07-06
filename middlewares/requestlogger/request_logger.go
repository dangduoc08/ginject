package requestlogger

import (
	"time"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

type RequestLogger struct {
	common.Logger
}

func (instance RequestLogger) Use(c *ctx.Context, next ctx.Next) {
	_, _ = c.Broker.Subscribe(ctx.RequestFinished, func(m *broker.Message) {
		newC := m.Payload.(*ctx.Context)

		var msg string
		switch newC.GetType() {
		case ctx.HTTPType:
			msg = newC.URL.String()
		case ctx.WSType:
			msg = c.GetWSConfig().Location.Path
		default:
			return
		}

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
