package config

import (
	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/ctx"
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

func (configService *ConfigService) Transform(s any) (any, []ctx.FieldLevel) {
	conf := map[string][]string{}
	for key, val := range configService.Config {
		conf[key] = append(conf[key], val.(string))
	}
	return ctx.BindStrArr(conf, &[]ctx.FieldLevel{}, s)
}
