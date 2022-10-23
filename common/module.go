package common

import (
	"log"
	"reflect"

	"github.com/dangduoc08/gooh"
)

type Module struct {
	Imports     []Module
	Providers   []Provider
	Exports     []Provider
	Controllers []Controller
}

func (module *Module) Create(app gooh.App) {
	providerMap := map[string]Provider{}
	moduleName := reflect.TypeOf(module)
	routerField := "Routers"

	for i, provider := range module.Providers {
		k := reflect.TypeOf(provider).Name()
		module.Providers[i] = provider.NewProvider()
		providerMap[k] = module.Providers[i]
	}

	for i, controller := range module.Controllers {
		copyController := reflect.New(reflect.TypeOf(controller))

		for j := 0; j < reflect.TypeOf(controller).NumField(); j++ {
			k := reflect.TypeOf(controller).Field(j).Name

			if providerMap[k] != nil && k != routerField {
				copyController.Elem().Field(j).Set(reflect.ValueOf(providerMap[k]))
			} else {
				if k == routerField {
					continue
				}
				log.Default().Fatalf("Can't resolve dependencies of the %v provider. Please make sure that the argument dependency at index [%v] is available in the %v context\n", k, j, moduleName)
			}
		}

		module.Controllers[i] = copyController.Interface().(Controller).NewController()

		for pattern, handler := range reflect.ValueOf(module.Controllers[i]).FieldByName(routerField).Interface().(map[string]gooh.Handler) {
			app.Route.Add(pattern, handler)
		}
	}
}
