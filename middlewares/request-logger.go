package middlewares

import (
	"fmt"
	"time"

	"github.com/dangduoc08/gooh/ctx"
)

func RequestLogger(ctx *ctx.Context) {
	nano := time.Now().Nanosecond()
	ctx.Event.On("finish", func(args ...interface{}) {
		fmt.Println("log-request", time.Now().Nanosecond()-nano)
	})

	ctx.Next()
}
