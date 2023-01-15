package config

import (
	"github.com/dangduoc08/gooh/core"
)

type Configer interface {
	Get(string) string
}

type ConfigProvider struct {
	Config map[string]string
}

func (configProvider ConfigProvider) Inject() core.Provider {
	return configProvider
}

func (configProvider *ConfigProvider) Get(k string) string {
	return k
}
