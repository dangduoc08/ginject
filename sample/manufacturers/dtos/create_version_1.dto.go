package dtos

import (
	"fmt"

	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
)

type CREATE_VERSION_1_Data_Body_TO struct {
	Name   string   `bind:"name"`
	Scores []string `bind:"scores"`
}

type CREATE_VERSION_1_Body_DTO struct {
	Data CREATE_VERSION_1_Data_Body_TO `bind:"data"`
}

func (instance CREATE_VERSION_1_Body_DTO) Transform(body ginject.Body, medata common.ArgumentMetadata) any {
	fmt.Println("[Module] CREATE_VERSION_1_Body dto")
	dto, _ := body.Bind(instance)

	return dto
}
