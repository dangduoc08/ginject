package middlewares

import (
	"github.com/dangduoc08/gooh/context"
)

func CORS() func(*context.Context) {
	return func(ctx *context.Context) {
		ctx.ResponseWriter.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.ResponseWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		ctx.ResponseWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		ctx.Next()
	}
}
