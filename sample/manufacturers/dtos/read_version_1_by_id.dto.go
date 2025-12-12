package dtos

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
)

type READ_BY_id_VERSION_1_Query_DTO struct {
	ID int `bind:"id"`
}

func (instance READ_BY_id_VERSION_1_Query_DTO) Transform(param ginject.Param, medata common.ArgumentMetadata) any {
	fmt.Println("[Module] READ_BY_id_VERSION_1_Query dto")
	dto, _ := param.Bind(instance)

	return dto
}
