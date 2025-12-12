package shared

import (
	"fmt"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/modules/config"
)

type AuthenticationGuard struct {
	common.Guard
	common.Logger
	config.ConfigService

	AuthKey    string
	AuthSecret string
}

func (instance AuthenticationGuard) NewGuard() AuthenticationGuard {
	instance.AuthKey = instance.ConfigService.Get("AUTH_KEY").(string)
	instance.AuthSecret = instance.ConfigService.Get("AUTH_SECRET").(string)

	return instance
}

func (instance AuthenticationGuard) CanActivate(ctx *ctx.Context) bool {
	fmt.Println("[Module] Authentication guard")

	reqSecret := ctx.Header().Get(instance.AuthKey)

	return reqSecret == instance.AuthSecret
}
