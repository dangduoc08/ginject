package task

import "github.com/dangduoc08/gooh/core"

var TaskModule = func() *core.Module {
	taskProvider := TaskProvider{}

	return core.ModuleBuilder().
		Providers(taskProvider).
		Exports(taskProvider).
		Build()
}()
