package friend

import (
	"github.com/dangduoc08/gooh/core"
)

var FriendModule = func() *core.Module {
	friendController := FriendController{}
	friendProvider := FriendProvider{}

	module := core.ModuleBuilder().
		Controllers(
			friendController,
		).
		Providers(
			friendProvider,
		).
		Exports(
			friendProvider,
		).
		Build()

	return module
}()
