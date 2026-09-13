package dto

import (
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

type PaginationDTO struct {
	Page  int `bind:"page"`
	Limit int `bind:"limit"`
}

func (paginationDTO PaginationDTO) Transform(query ctx.Query, arg common.ArgumentMetadata) any {
	bound, _ := query.Bind(paginationDTO)
	dto := bound.(PaginationDTO)

	if dto.Page < 1 {
		dto.Page = defaultPage
	}

	if dto.Limit < 1 {
		dto.Limit = defaultLimit
	} else if dto.Limit > maxLimit {
		dto.Limit = maxLimit
	}

	return dto
}

func (paginationDTO PaginationDTO) Skip() int {
	return (paginationDTO.Page - 1) * paginationDTO.Limit
}
