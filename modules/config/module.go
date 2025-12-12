package config

import (
	"os"
	"strings"

	"github.com/dangduoc08/ginject/core"
)

type (
	ConfigLoadFn   = func() map[string]any
	ConfigHookFn   = func(ConfigService)
	ConfigOnInitFn = func()
)

type ConfigModuleOptions struct {
	IsGlobal          bool
	IsIgnoreEnvFile   bool
	IsOverride        bool
	IsExpandVariables bool
	ENVFilePaths      []string
	Loads             []ConfigLoadFn
	Hooks             []ConfigHookFn
	OnInit            ConfigOnInitFn
}

func loadConfigOptions(opts *ConfigModuleOptions) *ConfigModuleOptions {
	if opts == nil {
		opts = &ConfigModuleOptions{}
	}

	envFilePaths := opts.ENVFilePaths
	if len(envFilePaths) == 0 {
		defaultPath, err := os.Getwd()
		if err != nil {
			panic(err)
		}
		defaultENVPath := defaultPath + "/.env"
		if _, err = os.Stat(defaultENVPath); err == nil {
			envFilePaths = []string{defaultENVPath}
		}
	}

	return &ConfigModuleOptions{
		IsGlobal:          opts.IsGlobal,
		IsIgnoreEnvFile:   opts.IsIgnoreEnvFile,
		IsOverride:        opts.IsOverride,
		IsExpandVariables: opts.IsExpandVariables,
		ENVFilePaths:      envFilePaths,
		Loads:             opts.Loads,
		Hooks:             opts.Hooks,
		OnInit:            opts.OnInit,
	}
}

func loadDotENV(path string, isExpandVariables bool) map[string]any {
	dotENVMap := make(map[string]any)
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	// prevent last index won't be appended into value
	data = append(data, newline)
	dotENV := &DotENV{
		data:   data,
		envMap: dotENVMap,
	}
	dotENV.Unmarshal()

	if isExpandVariables {
		for key, value := range dotENVMap {
			dotENVMap[key] = parseParamsToValue(key, value.(string), dotENVMap)
		}
	}

	return dotENVMap
}

func loadOSEnv() map[string]any {
	osENVMap := make(map[string]any)
	env := os.Environ()
	for _, v := range env {
		envArr := strings.Split(v, "=")
		if len(envArr) > 1 {
			osENVMap[envArr[0]] = envArr[1]
		}
	}

	return osENVMap
}

func mergeIntoOSENV(osENV map[string]any, isOverride bool, envs ...map[string]any) {
	for _, env := range envs {
		for key, value := range env {

			// key already set in machine
			// and not allow to override
			if osENV[key] != nil && !isOverride {
				continue
			}
			osENV[key] = value
		}
	}
}

func Register(opts *ConfigModuleOptions) *core.Module {
	configOptions := loadConfigOptions(opts)
	osENVMap := loadOSEnv()
	envs := []map[string]any{}

	if !configOptions.IsIgnoreEnvFile || len(opts.ENVFilePaths) > 0 {
		for _, path := range configOptions.ENVFilePaths {
			dotENVMap := loadDotENV(path, opts.IsExpandVariables)
			envs = append(envs, dotENVMap)
		}
	}

	if len(configOptions.Loads) > 0 {
		for _, loadCustomENV := range configOptions.Loads {
			customENVMap := loadCustomENV()
			customENVMap = flatten(customENVMap, make(map[string]any), "").(map[string]any)
			envs = append(envs, customENVMap)
		}
	}

	mergeIntoOSENV(osENVMap, configOptions.IsOverride, envs...)
	configService := ConfigService{osENVMap}

	if len(configOptions.Hooks) > 0 {
		for _, hookFn := range configOptions.Hooks {
			hookFn(configService)
			configService.Config = osENVMap
		}
	}

	module := core.ModuleBuilder().
		Providers(configService).
		Build()

	module.IsGlobal = configOptions.IsGlobal
	module.OnInit = configOptions.OnInit
	return module
}
