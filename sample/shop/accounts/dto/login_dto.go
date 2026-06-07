package dto

import (
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

// LoginDTO is the validated payload for starting a session.
//
//	POST /sessions
//	body: { "email": string, "password": string }
type LoginDTO struct {
	Email    string `bind:"email"`
	Password string `bind:"password"`
}

// Transform binds the request body into a LoginDTO and validates it,
// panicking with a BadRequestException when a field is missing.
func (loginDTO LoginDTO) Transform(body ctx.Body, arg common.ArgumentMetadata) any {
	bound, _ := body.Bind(loginDTO)
	dto := bound.(LoginDTO)

	dto.Email = strings.TrimSpace(dto.Email)

	if dto.Email == "" || dto.Password == "" {
		panic(exception.BadRequestException("email and password are required"))
	}

	return dto
}
