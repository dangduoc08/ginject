package global

import (
	"fmt"

	"github.com/dangduoc08/gooh"
)

func Middleware(ctx gooh.Context) {
	fmt.Println("Global Middleware involke")
	ctx.Next()
}
