package messenger

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
)

type MessengerPayloadDTO struct {
	EventName string `bind:"eventName"`
	Message   string `bind:"message"`
}

func (dto MessengerPayloadDTO) Transform(
	payload gooh.WSPayload,
	metadata common.ArgumentMetadata,
) any {
	return payload.Bind(dto)
}
