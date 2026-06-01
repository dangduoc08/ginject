package shared

import (
	"fmt"
	"time"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

type RequestLogger struct {
	common.Logger
}

func (instance RequestLogger) Use(c ginject.Context, next ginject.Next) {
	fmt.Println("[Global] RequestLogger middleware")
	_, _ = c.Broker.Subscribe(ctx.REQUEST_FINISHED, func(m *broker.Message) {
		newC := m.Payload.(*ctx.Context)
		requestType := newC.GetType()
		responseTime := time.Now().UnixMilli() - newC.Timestamp.UnixMilli()

		switch requestType {
		case ctx.HTTPType:
			instance.Info(
				newC.URL.String(),
				"Method", newC.Method,
				"Status", newC.Code,
				"Time", fmt.Sprintf("%v ms", responseTime),
				"Protocol", newC.Proto,
				"User-Agent", newC.UserAgent(),
				ctx.REQUEST_ID, newC.GetID(),
			)
		case ctx.WSType:
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
