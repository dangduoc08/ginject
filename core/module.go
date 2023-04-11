package core

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/dangduoc08/gooh/common"
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
	router         *routing.Route
	staticModules  []*Module
	dynamicModules []any
	providers      []Provider
	exports        []Provider
	controllers    []Controller

	Middleware *Middleware
	IsGlobal   bool
	OnInit     func()
}

func (m *Module) injectMainModules() {

	// append module pointer to a list of modules
	// which injected from the main function
	modulesInjectedFromMain = append(modulesInjectedFromMain, reflect.ValueOf(m).Pointer())
}

func (m *Module) injectGlobalProviders() {
	for _, provider := range m.exports {

		// generate a unique key for the provider
		globalProviders[genProviderKey(provider)] = provider
	}
}

func (m *Module) NewModule() *Module {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	if m.singleInstance == nil {
		m.singleInstance = m
		if m.OnInit != nil {
			m.OnInit()
		}

		// first injection always from main module
		// invoked by create function.
		// only modules injected by main module
		// are able to use controllers
		if len(modulesInjectedFromMain) == 0 {
			m.injectMainModules()

			// main module's provider
			// alway inject globally
			m.injectGlobalProviders()

			for _, staticModule := range m.staticModules {
				staticModule.injectMainModules()

				if staticModule.IsGlobal {
					staticModule.injectGlobalProviders()
				}
			}
		}

		// inject static modules
		for _, staticModule := range m.staticModules {

			// recursion injection
			injectModule := staticModule.NewModule()

			// only import providers which exported
			if len(injectModule.exports) > 0 {
				m.providers = append(m.providers, injectModule.exports...)
			}

			m.router.Group("/", injectModule.router)
		}

		// inject local providers
		var injectedProviders map[string]Provider = make(map[string]Provider)
		for _, provider := range m.providers {
			injectedProviders[genProviderKey(provider)] = provider
		}

		// inject dynamic modules
		for _, dynamicModule := range m.dynamicModules {
			staticModule := createStaticModuleFromDynamicModule(dynamicModule, injectedProviders)

			// dynamic modules will be treated
			// as global module
			// hence dynamic module's controllers
			// always are injected
			staticModule.injectMainModules()

			if staticModule.IsGlobal {
				staticModule.injectGlobalProviders()
			}

			injectModule := staticModule.NewModule()

			// only import providers which exported
			if len(injectModule.exports) > 0 {
				m.providers = append(m.providers, injectModule.exports...)
			}

			m.router.Group("/", injectModule.router)
		}

		// inject local providers
		// from dynamic modules
		// line 94 already inject (not bug)
		for _, provider := range m.providers {
			injectedProviders[genProviderKey(provider)] = provider
		}

		// inject providers into providers
		for i, provider := range m.providers {
			providerType := reflect.TypeOf(provider)
			providerValue := reflect.ValueOf(provider)
			newProvider := reflect.New(providerType)
			providerKey := providerType.String()

			// injected providers inside providers
			// can be injected through global modules
			// or through imported modules
			for j := 0; j < providerType.NumField(); j++ {
				providerField := providerType.Field(j)
				providerFieldType := providerField.Type
				providerFieldKey := providerFieldType.String()
				providerFieldName := providerField.Name

				if utils.StrIsLower(providerFieldName[0:1])[0] {
					panic(fmt.Errorf(
						utils.FmtRed(
							"can't set value to unexported %v field of the %v provider",
							providerFieldName,
							providerType.Name(),
						),
					))
				}

				// inject provider priorities
				// local inject
				// global inject
				// inner packages
				// resolve dependencies error
				if providerFieldKey != "" && injectedProviders[providerFieldKey] != nil {
					newProvider.Elem().Field(j).Set(reflect.ValueOf(injectedProviders[providerFieldKey]))
				} else if providerFieldKey != "" && globalProviders[providerFieldKey] != nil {
					newProvider.Elem().Field(j).Set(reflect.ValueOf(globalProviders[providerFieldKey]))
				} else if !isInjectedProvider(providerFieldType) {
					newProvider.Elem().Field(j).Set(providerValue.Field(j))
				} else {
					panic(fmt.Errorf(
						utils.FmtRed(
							"can't resolve dependencies of the %v provider. Please make sure that the argument dependency at index [%v] is available in the %v provider",
							providerFieldKey,
							j,
							providerType.Name(),
						),
					))
				}
			}

			if providerInjectCheck[providerKey] == nil {
				providerInjectCheck[providerKey] = newProvider.Interface().(Provider).NewProvider()
			}

			m.providers[i] = providerInjectCheck[providerKey]
			injectedProviders[providerKey] = providerInjectCheck[providerKey]
		}

		// inject providers into controllers
		if utils.ArrIncludes(modulesInjectedFromMain, reflect.ValueOf(m).Pointer()) {
			for i, controller := range m.controllers {
				controllerType := reflect.TypeOf(controller)
				controllerValue := reflect.ValueOf(controller)
				newControllerType := reflect.New(controllerType)

				for j := 0; j < controllerType.NumField(); j++ {
					controllerField := controllerType.Field(j)
					controllerFieldType := controllerField.Type
					controllerFieldNameKey := controllerField.Name

					if utils.StrIsLower(controllerFieldNameKey[0:1])[0] {
						panic(fmt.Errorf(
							utils.FmtRed(
								"can't set value to unexported %v field of the %v controller",
								controllerFieldNameKey,
								controllerType.Name(),
							),
						))
					}

					injectProviderKey := controllerType.Field(j).Type.String()
					isUnneededInject := utils.ArrIncludes(noInjectedFields, injectProviderKey)

					if injectedProviders[injectProviderKey] != nil && !isUnneededInject {
						newControllerType.Elem().Field(j).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
					} else if globalProviders[injectProviderKey] != nil && !isUnneededInject {
						newControllerType.Elem().Field(j).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
					} else if !isInjectedProvider(controllerFieldType) {
						newControllerType.Elem().Field(j).Set(controllerValue.Field(j))
					} else {
						if isUnneededInject {
							continue
						}
						panic(fmt.Errorf(
							utils.FmtRed(
								"can't resolve dependencies of the %v provider. Please make sure that the argument dependency at index [%v] is available in the %v controller",
								injectProviderKey,
								j,
								controllerType.Name(),
							),
						))
					}
				}

				m.controllers[i] = newControllerType.Interface().(Controller).NewController()

				configInclusion := []RouteConfig{}
				for pattern := range reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[0]).Interface().(common.Rest).RouterMap {
					method, path := routing.SplitRoute(pattern)
					configInclusion = append(configInclusion, RouteConfig{
						Method: method,
						Path:   path,
					})
				}
				m.Middleware.include(configInclusion).add()

				// apply module bound middlewares
				for _, middlewareMap := range m.Middleware.middlewareMapArr {
					for path, configs := range middlewareMap {
						m.router.For(path, configs.methods)(configs.handlers...)
					}
				}

				// add main handler
				for pattern, handler := range reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[0]).Interface().(common.Rest).RouterMap {
					if err := isInjectableHandler(handler); err != nil {
						panic(utils.FmtRed(err.Error()))
					}

					method, _ := routing.SplitRoute(pattern)
					m.router.AddInjectableHandler(pattern, method, handler)
				}
			}
		}
	}

	return m
}
