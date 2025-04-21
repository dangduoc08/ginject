package shared

import (
	"fmt"
	"time"

	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
	"github.com/dangduoc08/gogo/ctx"
)

type RequestLogger struct {
	common.Logger
}

func (instance RequestLogger) Use(c gogo.Context, next gogo.Next) {
	fmt.Println("[Global] RequestLogger middleware")
	c.Event.On(ctx.REQUEST_FINISHED, func(args ...any) {
		newC := args[0].(*ctx.Context)
		requestType := newC.GetType()
		responseTime := time.Now().UnixMilli() - newC.Timestamp.UnixMilli()

		if requestType == ctx.HTTPType {
			instance.Info(
				newC.URL.String(),
				"Method", newC.Method,
				"Status", newC.Code,
				"Time", fmt.Sprintf("%v ms", responseTime),
				"Protocol", newC.Request.Proto,
				"User-Agent", newC.UserAgent(),
				ctx.REQUEST_ID, newC.GetID(),
			)
		} else if requestType == ctx.WSType {
			instance.Info(
				newC.WS.Message.Event,
				"Time", fmt.Sprintf("%v ms", responseTime),
				"Subprotocol", newC.WS.GetSubprotocol(),
				"User-Agent", newC.UserAgent(),
			)
		}
	})

	next()
}
