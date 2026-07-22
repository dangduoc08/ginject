package accounts

import (
	"strings"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/sample/shop/accounts/dto"
)

// SessionsController models login/logout as creating and deleting a
// session resource: POST starts one (login), DELETE ends the caller's
// current one (logout).
type SessionsController struct {
	common.HTTP
	common.Guard

	UserService
}

func (instance SessionsController) NewController() core.Controller {
	instance.BindGuard(AuthGuard{}, instance.DELETE_sessions)

	return instance
}

// CREATE_sessions logs a user in by exchanging email/password for a bearer
// token.
//
//	POST /sessions
//	body: { "email": string, "password": string }
func (instance SessionsController) CREATE_sessions(loginDTO dto.LoginDTO) ginject.Map {
	user, token, err := instance.UserService.Authenticate(loginDTO.Email, loginDTO.Password)
	if err != nil {
		panic(err)
	}

	return ginject.Map{
		"user":  user,
		"token": token,
	}
}

// DELETE_sessions logs the caller out by revoking the bearer token sent in
// the Authorization header. Requires AuthGuard.
//
//	DELETE /sessions
//	header: Authorization: Bearer <token>
func (instance SessionsController) DELETE_sessions(header ginject.Header) ginject.Map {
	token := strings.TrimPrefix(header.Get("Authorization"), "Bearer ")
	instance.UserService.Logout(token)

	return ginject.Map{
		"message": "logged out",
	}
}
