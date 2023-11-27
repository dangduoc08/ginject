package config

import (
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/core"
)

type ConfigService struct {
	Config map[string]any
}

func (configService ConfigService) NewProvider() core.Provider {
	return configService
}

func (configService *ConfigService) Get(k string) any {
	return configService.Config[k]
}

func (configService *ConfigService) Set(k string, v any) {
	configService.Config[k] = v
}

func (configService *ConfigService) Transform(s any) any {
	conf := map[string][]string{}
	for key, val := range configService.Config {
		conf[key] = append(conf[key], val.(string))
	}
	return context.BindStrArr(conf, s)
}
