package messenger

import "github.com/dangduoc08/gooh/core"

var MessengerModule = func() *core.Module {
	messengerProvider := MessengerProvider{}
	messengerController := MessengerController{}

	return core.ModuleBuilder().
		Providers(messengerProvider).
		Controllers(messengerController).
		Build()
}()
