package accounts

import (
	"context"
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/ctx"
)

const currentUserKey core.WithValueKey = "shop.currentUser"

type AuthGuard struct {
	common.Guard
	UserService
}

func (instance AuthGuard) NewGuard() AuthGuard {
	return instance
}

func (instance AuthGuard) CanActivate(c *ctx.HTTPContext) bool {
	token := strings.TrimPrefix(c.Header().Get("Authorization"), "Bearer ")
	if token == "" {
		return false
	}

	user, ok := instance.UserService.UserBySession(token)
	if !ok {
		return false
	}

	c.Request = c.WithContext(context.WithValue(c.Context(), currentUserKey, user))

	return true
}

func CurrentUser(c *ctx.HTTPContext) User {
	user, _ := c.Context().Value(currentUserKey).(User)

	return user
}
