package dto

import (
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

type LoginDTO struct {
	Email    string `bind:"email"`
	Password string `bind:"password"`
}

func (loginDTO LoginDTO) Transform(body ctx.Body, arg common.ArgumentMetadata) any {
	bound, _ := body.Bind(loginDTO)
	dto := bound.(LoginDTO)

	dto.Email = strings.TrimSpace(dto.Email)

	if dto.Email == "" || dto.Password == "" {
		panic(exception.BadRequestException("email and password are required"))
	}

	return dto
}
