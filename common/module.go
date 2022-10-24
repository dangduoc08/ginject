package common

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type Module struct {
	// sync.Mutex
	singleInstance *Module

	Imports      []Module
	Providers    []Provider
	Exports      []Provider
	Controllers  []Controller
	ModuleRouter *routing.Route
}

func (module *Module) GetInstance() *Module {
	if module.singleInstance == nil {
		// module.Mutex.Lock()
		// defer module.Mutex.Unlock()

		module.ModuleRouter = routing.NewRoute()
		noInjectedFields := []string{
			"Control",
			"common.Control",
		}

		for _, subModule := range module.Imports {
			injectModule := subModule.GetInstance()

			if len(injectModule.Exports) > 0 {
				module.Providers = append(module.Providers, injectModule.Exports...)
			}

			module.ModuleRouter.Group("/", injectModule.ModuleRouter)
		}

		providerMap := map[string]Provider{}
		for i, provider := range module.Providers {
			providerKey := reflect.TypeOf(provider).String()
			module.Providers = append(module.Providers, provider.NewProvider())
			providerMap[providerKey] = module.Providers[i]
		}

		for i, controller := range module.Controllers {
			controllerType := reflect.TypeOf(controller)
			copyController := reflect.New(controllerType)

			for j := 0; j < controllerType.NumField(); j++ {
				injectProviderKey := controllerType.Field(j).Type.String()
				fieldName := controllerType.Field(j).Name
				if fieldName[0:1] == strings.ToLower(fieldName[0:1]) {
					panic(fmt.Errorf("can't set value to unexported %v field of the %v controller", fieldName, controllerType.Name()))
				}

				isUnneededInject := utils.ArrIncludes(noInjectedFields, injectProviderKey)

				if providerMap[injectProviderKey] != nil && !isUnneededInject {
					copyController.Elem().Field(j).Set(reflect.ValueOf(providerMap[injectProviderKey]))
				} else {
					if isUnneededInject {
						continue
					}
					panic(fmt.Errorf("can't resolve dependencies of the %v provider. Please make sure that the argument dependency at index [%v] is available in the %v controller of context module", injectProviderKey, j, controllerType.Name()))
				}
			}

			module.Controllers[i] = copyController.Interface().(Controller).NewController()

			for pattern, handlers := range reflect.ValueOf(module.Controllers[i]).FieldByName(noInjectedFields[0]).Interface().(Control).routers {
				module.ModuleRouter.Add(pattern, handlers...)
			}
		}

		module.singleInstance = module
	}

	return module
}
