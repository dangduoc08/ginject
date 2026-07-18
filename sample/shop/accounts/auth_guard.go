package accounts

import (
	"context"
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/ctx"
)

// currentUserKey is the context.Context key AuthGuard stores the
// authenticated User under, so handlers can retrieve it without re-resolving
// the bearer token.
const currentUserKey core.WithValueKey = "shop.currentUser"

// AuthGuard requires a valid "Authorization: Bearer <token>" header. On
// success it attaches the resolved User to the request context so handlers
// behind it can read it via CurrentUser.
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

// CurrentUser returns the User attached to c by AuthGuard. It must only be
// called from handlers placed behind AuthGuard.
func CurrentUser(c *ctx.HTTPContext) User {
	user, _ := c.Context().Value(currentUserKey).(User)

	return user
}
