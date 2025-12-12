package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/modules/config"
)

type RateLimiterGuard struct {
	common.Guard
	common.Logger
	config.ConfigService
}

func (instance RateLimiterGuard) CanActivate(ctx *ctx.Context) bool {
	fmt.Println("[Global] RateLimiter guard")

	return true
}
