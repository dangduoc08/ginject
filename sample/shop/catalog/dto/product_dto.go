package dto

import (
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

type ProductDTO struct {
	Name  string  `bind:"name"`
	Price float64 `bind:"price"`
}

func (productDTO ProductDTO) Transform(body ctx.Body, arg common.ArgumentMetadata) any {
	bound, _ := body.Bind(productDTO)
	dto := bound.(ProductDTO)

	dto.Name = strings.TrimSpace(dto.Name)

	if dto.Name == "" {
		panic(exception.BadRequestException("name is required"))
	}
	if dto.Price <= 0 {
		panic(exception.BadRequestException("price must be a positive number"))
	}

	return dto
}
