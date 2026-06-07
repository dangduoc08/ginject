package dto

import (
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

// CategoryDTO is the validated payload for creating or renaming a category.
//
//	body: { "name": string }
type CategoryDTO struct {
	Name string `bind:"name"`
}

// Transform binds the request body into a CategoryDTO and validates it,
// panicking with a BadRequestException when the name is missing.
func (categoryDTO CategoryDTO) Transform(body ctx.Body, arg common.ArgumentMetadata) any {
	bound, _ := body.Bind(categoryDTO)
	dto := bound.(CategoryDTO)

	dto.Name = strings.TrimSpace(dto.Name)

	if dto.Name == "" {
		panic(exception.BadRequestException("name is required"))
	}

	return dto
}
