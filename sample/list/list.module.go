package list

import (
	"fmt"

	"github.com/dangduoc08/gooh"
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

	module.Middleware.
		Apply(
			func(c gooh.Context) {
				fmt.Println("ListModule Middleware1")
				c.Next()
			},
			func(c gooh.Context) {
				fmt.Println("ListModule Middleware2")
				c.Next()
			},
		).
		Exclude([]any{
			listController.READ_lists,
		})

	return module
}()
