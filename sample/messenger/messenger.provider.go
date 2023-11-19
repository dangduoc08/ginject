package messenger

import "github.com/dangduoc08/gooh/core"

type MessengerProvider struct{}

func (messengerProvider MessengerProvider) NewProvider() core.Provider {
	return messengerProvider
}
