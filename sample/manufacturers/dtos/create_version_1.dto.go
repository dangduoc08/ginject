package dtos

import (
	"fmt"

	"github.com/dangduoc08/gogo"
	"github.com/dangduoc08/gogo/common"
)

type CREATE_VERSION_1_DTO struct {
	Name string `bind:"name"`
}

func (instance CREATE_VERSION_1_DTO) Transform(body gogo.Body, medata common.ArgumentMetadata) any {
	fmt.Println("[Module] CREATE_VERSION_1 dto")
	dto, _ := body.Bind(instance)

	return dto
}
