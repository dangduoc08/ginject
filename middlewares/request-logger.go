package middlewares

import (
	"log"
	"time"

	"github.com/dangduoc08/gooh/ctx"
)

func RequestLogger(c *ctx.Context) {
	c.Event.On(ctx.REQUEST_FINISHED, func(args ...interface{}) {
		responseTime := time.Now().UnixMilli() - c.Timestamp.UnixMilli()
		log.Printf("%v %v %v %v - %v ms", c.Method, c.UserAgent(), c.URL, c.Code, responseTime)
	})

	c.Next()
}
