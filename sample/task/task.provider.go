package task

import "github.com/dangduoc08/gooh/core"

type TaskProvider struct{}

func (taskProvider TaskProvider) NewProvider() core.Provider {
	return taskProvider
}
