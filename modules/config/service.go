package config

import (
	"github.com/dangduoc08/gooh/core"
)

type ConfigService struct {
	Config map[string]any
}

func (configService ConfigService) Inject() core.Provider {
	return configService
}

func (configService *ConfigService) Get(k string) any {
	return configService.Config[k]
}
