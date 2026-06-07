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

// PaginationDTO is the validated query payload for list endpoints.
//
//	query: ?page=<int>&limit=<int>
//
// Missing or non-positive values fall back to defaults; limit is capped at
// maxLimit so a caller can't force the server to scan unbounded pages.
type PaginationDTO struct {
	Page  int `bind:"page"`
	Limit int `bind:"limit"`
}

// Transform binds the request query into a PaginationDTO and normalizes it —
// unlike body DTOs, out-of-range pagination values are clamped rather than
// rejected, since "page=0" or "limit=9999" has an obvious sane interpretation.
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

// Skip returns the number of items to skip to reach this page.
func (paginationDTO PaginationDTO) Skip() int {
	return (paginationDTO.Page - 1) * paginationDTO.Limit
}
