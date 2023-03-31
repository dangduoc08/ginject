package middlewares

import (
	"log"
	"time"

	"github.com/dangduoc08/gooh/context"
)

func RequestLogger(c *context.Context) {
	c.Event.On(context.REQUEST_FINISHED, func(args ...any) {
		responseTime := time.Now().UnixMilli() - c.Timestamp.UnixMilli()
		log.Printf("%v %v %v %v - %v ms", c.Method, c.UserAgent(), c.URL, c.Code, responseTime)
	})

	c.Next()
}
