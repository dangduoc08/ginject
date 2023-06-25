package dtos

import (
	"fmt"

	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/modules/config"
)

type UpdateReviewsByReviewIDOfByBookIDDTO struct {
	gooh.Query
	ConfigService config.ConfigService
	Name          string
	Age           int
}

func (dto UpdateReviewsByReviewIDOfByBookIDDTO) Transform(value any, metadata common.ArgumentMetadata) any {
	fmt.Println("dto.ConfigService", dto.ConfigService)

	return dto
}
