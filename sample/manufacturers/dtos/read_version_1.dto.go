package dtos

import (
	"fmt"

	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
)

type READ_VERSION_1_Query_DTO struct {
	Limit  uint `bind:"limit"`
	Offset uint `bind:"offset"`
}

func (instance READ_VERSION_1_Query_DTO) Transform(query gogo.Query, medata common.ArgumentMetadata) any {
	fmt.Println("[Module] READ_VERSION_1_Query dto")
	dto, _ := query.Bind(instance)

	return dto
}
