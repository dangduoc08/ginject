package dto

import (
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

// UserDTO is the validated payload for account registration.
//
//	POST /users
//	body: { "email": string, "name": string, "password": string }
type UserDTO struct {
	Email    string `bind:"email"`
	Name     string `bind:"name"`
	Password string `bind:"password"`
}

// Transform binds the request body into a UserDTO and validates it,
// panicking with a BadRequestException when a field is missing or malformed.
func (userDTO UserDTO) Transform(body ctx.Body, arg common.ArgumentMetadata) any {
	bound, _ := body.Bind(userDTO)
	dto := bound.(UserDTO)

	dto.Email = strings.TrimSpace(dto.Email)
	dto.Name = strings.TrimSpace(dto.Name)

	if dto.Email == "" || dto.Name == "" || dto.Password == "" {
		panic(exception.BadRequestException("email, name and password are required"))
	}
	if !strings.Contains(dto.Email, "@") {
		panic(exception.BadRequestException("email must be a valid email address"))
	}
	if len(dto.Password) < 8 {
		panic(exception.BadRequestException("password must be at least 8 characters long"))
	}

	return dto
}
