package messenger

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/core"
)

type MessengerController struct {
	common.WS
	MessengerProvider MessengerProvider
	Logger            common.Logger
}

func (messengerController MessengerController) NewController() core.Controller {
	messengerController.Subprotocol("messenger")

	return messengerController
}

func (messengerController MessengerController) SUBSCRIBE_sendMessage(
	messengerPayloadDTO MessengerPayloadDTO,
) (string, any) {

	return messengerPayloadDTO.EventName, gooh.Map{
		"messenger": messengerPayloadDTO.Message,
	}
}
