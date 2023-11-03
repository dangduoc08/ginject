package list

import (
	"github.com/dangduoc08/gooh/core"
	"github.com/dangduoc08/gooh/sample/task"
)

var ListModule = func() *core.Module {
	listController := ListController{}
	listProvider := ListProvider{}

	module := core.ModuleBuilder().
		Imports(
			task.TaskModule,
		).
		Controllers(
			listController,
		).
		Providers(
			listProvider,
		).
		Exports(
			listProvider,
		).
		Build()

	return module
}()
