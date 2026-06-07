package dto

import (
	"strings"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

// ProductDTO is the validated payload for creating or updating a product.
//
//	body: { "name": string, "price": number }
type ProductDTO struct {
	Name  string  `bind:"name"`
	Price float64 `bind:"price"`
}

// Transform binds the request body into a ProductDTO and validates it,
// panicking with a BadRequestException when the name is missing or the
// price is not a positive number.
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
