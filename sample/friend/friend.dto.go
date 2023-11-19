package friend

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
)

type ReadFriendsParamDTO struct {
	UserID int `bind:"userID"`
}

func (dto ReadFriendsParamDTO) Transform(
	param gooh.Param,
	metadata common.ArgumentMetadata,
) any {
	return param.Bind(dto)
}
