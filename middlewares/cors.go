package middlewares

import "github.com/dangduoc08/gogo/ctx"

func CORS() func(*ctx.Context) {
	return func(c *ctx.Context) {
		c.ResponseWriter.Header().Set("Access-Control-Allow-Origin", "*")
		c.ResponseWriter.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.ResponseWriter.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		c.Next()
	}
}
