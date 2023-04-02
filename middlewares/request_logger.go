package middlewares

import (
	"log"
	"time"

	"github.com/dangduoc08/gooh/context"
)

func RequestLogger(c *context.Context) {
	c.Event.On(context.REQUEST_FINISHED, func(args ...any) {
		newC := args[0].(*context.Context)
		responseTime := time.Now().UnixMilli() - newC.Timestamp.UnixMilli()
		log.Printf("%v %v %v %v - %v ms", newC.Method, newC.UserAgent(), newC.URL, newC.Code, responseTime)
	})

	c.Next()
}
