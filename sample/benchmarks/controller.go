package benchmarks

import (
	"github.com/dangduoc08/ginject"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/modules/httpclient"
)

type Controller struct {
	common.REST
	common.WS
	httpclient.ClientService
}

func (instance Controller) NewController() core.Controller {

	return instance
}

func (instance Controller) READ_ping() ginject.Map {
	return ginject.Map{
		"message": "Hello, World!",
	}
}

func (instance Controller) SUBSCRIBE_chat() ginject.Map {
	return ginject.Map{
		"message": "Hello, World!",
	}
}
