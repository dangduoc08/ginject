package core

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

var modulesInjectedFromMain []uintptr
var globalProviders map[string]Provider = make(map[string]Provider)
var providerInjectCheck map[string]Provider = make(map[string]Provider)
var noInjectedFields = []string{
	"Rest",
	"core.Rest",
}

type Module struct {
	*sync.Mutex
	singleInstance *Module
	IsGlobal       bool
	Imports        []*Module
	Providers      []Provider
	Exports        []Provider
	Controllers    []Controller
	Router         *routing.Route
	OnInit         func()
}

func (m *Module) Inject() *Module {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if m.singleInstance == nil {
		m.singleInstance = m
		if m.OnInit != nil {
			m.OnInit()
		}

		// first inject always from main module
		// invoked by create function.
		// only modules injected by main module
		// able to use presenter
		if len(modulesInjectedFromMain) == 0 {
			for _, subModule := range m.Imports {
				modulesInjectedFromMain = append(modulesInjectedFromMain, reflect.ValueOf(m).Pointer())
				for _, subModule := range m.Imports {
					modulesInjectedFromMain = append(modulesInjectedFromMain, reflect.ValueOf(subModule).Pointer())
				}

				// module which IsGlobal = true can
				// global inject providers
				// submodule no need to import module
				if subModule.IsGlobal {
					for _, subProvider := range subModule.Providers {
						subProviderType := reflect.TypeOf(subProvider)
						subProviderKey := subProviderType.String()

						globalProviders[subProviderKey] = subProvider
					}
				}
			}
		}

		for _, subModule := range m.Imports {
			injectModule := subModule.Inject()

			// only import providers which exported
			if len(injectModule.Exports) > 0 {
				m.Providers = append(m.Providers, injectModule.Exports...)
			}

			m.Router.Group("/", injectModule.Router)
		}

		// local inject providers
		var injectedProviders map[string]Provider = make(map[string]Provider)
		for _, provider := range m.Providers {
			providerType := reflect.TypeOf(provider)
			providerKey := providerType.String()

			injectedProviders[providerKey] = provider
		}

		for i, provider := range m.Providers {
			providerType := reflect.TypeOf(provider)
			providerValue := reflect.ValueOf(provider)
			newProvider := reflect.New(providerType)
			providerKey := providerType.String()

			// injected providers inside providers
			// can be injected through global modules
			// or through imported modules
			for j := 0; j < providerType.NumField(); j++ {
				providerFieldNameKey := providerType.Field(j).Type.String()

				// inject provider priorities
				// local inject
				// global inject
				// inner packages
				if providerFieldNameKey != "" && injectedProviders[providerFieldNameKey] != nil {
					newProvider.Elem().Field(j).Set(reflect.ValueOf(injectedProviders[providerFieldNameKey]))
				} else if providerFieldNameKey != "" && globalProviders[providerFieldNameKey] != nil {
					newProvider.Elem().Field(j).Set(reflect.ValueOf(globalProviders[providerFieldNameKey]))
				} else {
					newProvider.Elem().Field(j).Set(providerValue.Field(j))
				}
			}

			if providerInjectCheck[providerKey] == nil {
				providerInjectCheck[providerKey] = newProvider.Interface().(Provider).Inject()
			}

			m.Providers[i] = providerInjectCheck[providerKey]
			injectedProviders[providerKey] = providerInjectCheck[providerKey]
		}
		if utils.ArrIncludes(modulesInjectedFromMain, reflect.ValueOf(m).Pointer()) {
			for i, controller := range m.Controllers {
				controllerType := reflect.TypeOf(controller)
				newControllerType := reflect.New(controllerType)

				for j := 0; j < controllerType.NumField(); j++ {
					fieldName := controllerType.Field(j).Name
					if utils.StrIsLower(fieldName[0:1])[0] {
						panic(fmt.Errorf("can't set value to unexported %v field of the %v controller", fieldName, controllerType.Name()))
					}

					injectProviderKey := controllerType.Field(j).Type.String()
					isUnneededInject := utils.ArrIncludes(noInjectedFields, injectProviderKey)

					if injectedProviders[injectProviderKey] != nil && !isUnneededInject {
						newControllerType.Elem().Field(j).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
					} else if globalProviders[injectProviderKey] != nil && !isUnneededInject {
						newControllerType.Elem().Field(j).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
					} else {
						if isUnneededInject {
							continue
						}
						panic(fmt.Errorf("can't resolve dependencies of the %v provider. Please make sure that the argument dependency at index [%v] is available in the %v controller", injectProviderKey, j, controllerType.Name()))
					}
				}

				m.Controllers[i] = newControllerType.Interface().(Controller).Inject()

				for pattern, handlers := range reflect.ValueOf(m.Controllers[i]).FieldByName(noInjectedFields[0]).Interface().(Rest).routerMap {
					m.Router.Add(pattern, handlers...)
				}
			}
		}
	}

	return m
}
