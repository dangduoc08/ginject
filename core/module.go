package core

import (
	"errors"
	"go/token"
	"reflect"
	"sync"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/utils"
)

var mainModulePtr uintptr
var modulesInjectedFromMain []uintptr
var injectedDynamicModules = make(map[uintptr]*Module)
var globalPrefixArr = []map[string][]string{}
var globalProviders map[string]Provider = make(map[string]Provider)
var globalInterfaces map[string]any = make(map[string]any)
var providerInjectCheck map[string]Provider = make(map[string]Provider)
var noInjectedFields = []string{
	"REST",
	"common.REST",
	"Guard",
	"common.Guard",
	"Interceptor",
	"common.Interceptor",
	"ExceptionFilter",
	"common.ExceptionFilter",
	"WS",
	"common.WS",
	"Middleware",
	"common.Middleware",
}
var injectableInterfaces = []string{
	"github.com/dangduoc08/ginject/common/common.Logger",
}

type Module struct {
	id       string
	prefixes []string

	*sync.Mutex
	singleInstance *Module
	staticModules  []*Module
	dynamicModules []any
	providers      []Provider
	controllers    []Controller

	IsGlobal bool
	OnInit   func()

	// store REST module exception filters
	RESTExceptionFilters []common.RESTLayer

	// store REST module middlewares
	RESTMiddlewares []common.RESTLayer

	// store REST module guards
	RESTGuards []common.RESTLayer

	// store REST module interceptors
	RESTInterceptors []common.RESTLayer

	// store REST main handlers
	RESTMainHandlers []common.RESTLayer

	// store WS module middlewares
	WSMiddlewares []struct {
		controllerName string
		Subprotocol    string
		EventName      string
		Handler        any
	}

	// store WS module guards
	WSGuards []struct {
		Subprotocol string
		EventName   string
		Handler     any
	}

	// store WS module interceptors
	WSInterceptors []struct {
		Subprotocol string
		EventName   string
		Handler     any
	}

	// store WS module exception filters
	WSExceptionFilters []struct {
		Subprotocol string
		EventName   string
		Handler     any
	}

	// store WS main handlers
	WSMainHandlers []struct {
		Subprotocol string
		EventName   string
		Handler     any
	}
}

func (m *Module) injectGlobalProviders() {
	for _, provider := range m.providers {

		// generate a unique key for the provider
		globalProviders[genProviderKey(provider)] = provider
	}
}

func (m *Module) Prefix(prefix string) *Module {
	m.prefixes = append([]string{routing.ToEndpoint(prefix)}, m.prefixes...)

	return m
}

func (m *Module) ID() string {
	return m.id
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
		if mainModulePtr == 0 {
			modulesInjectedFromMain = append(modulesInjectedFromMain, reflect.ValueOf(m).Pointer())
			mainModulePtr = reflect.ValueOf(m).Pointer()

			// main module's provider
			// alway inject globally
			m.injectGlobalProviders()

			// static modules which inject in main.go
			for _, staticModule := range m.staticModules {
				m.controllers = append(m.controllers, staticModule.controllers...)
				m.providers = append(m.providers, staticModule.providers...)

				// static modules which set as globally
				// must be injected in main module
				if staticModule.IsGlobal {
					staticModule.injectGlobalProviders()
				}
			}

			// dynamic modules which inject in main.go
			for _, dynamicModule := range m.dynamicModules {
				staticModule := createStaticModuleFromDynamicModule(dynamicModule)
				injectedDynamicModules[reflect.ValueOf(dynamicModule).Pointer()] = staticModule

				m.controllers = append(m.controllers, staticModule.controllers...)
				m.providers = append(m.providers, staticModule.providers...)

				// dynamic modules which set as globally
				// have to be injected in main module
				if staticModule.IsGlobal {
					staticModule.injectGlobalProviders()
				}
			}
		}

		// inject static modules
		for _, staticModule := range m.staticModules {

			// no need inject global here
			// since globally static modules
			// shoule be injected from main
			// to make it injectable

			// recursion injection
			injectModule := staticModule.NewModule()
			if len(injectModule.providers) > 0 {
				m.providers = append(injectModule.providers, m.providers...)
			}
			if len(injectModule.controllers) > 0 {
				m.controllers = append(injectModule.controllers, m.controllers...)
			}
			toUniqueControllers(m, &m.controllers)
		}

		// inject dynamic modules
		for _, dynamicModule := range m.dynamicModules {
			var staticModule *Module

			dynamicModulePtr := reflect.ValueOf(dynamicModule).Pointer()

			if storedInjectModule, ok := injectedDynamicModules[dynamicModulePtr]; ok {
				staticModule = storedInjectModule
			} else {
				staticModule = createStaticModuleFromDynamicModule(dynamicModule)
				injectedDynamicModules[dynamicModulePtr] = staticModule
			}

			injectModule := staticModule.NewModule()
			if len(injectModule.providers) > 0 {
				m.providers = append(injectModule.providers, m.providers...)
			}
			if len(injectModule.controllers) > 0 {
				m.controllers = append(injectModule.controllers, m.controllers...)
			}
			toUniqueControllers(m, &m.controllers)
		}

		// set module prefixes
		for _, controller := range m.controllers {
			globalPrefixArr = append(globalPrefixArr, map[string][]string{
				genControllerKey(m, controller): m.prefixes,
			})
		}

		// inject local providers
		// from static/dynamic modules
		var injectedProviders map[string]Provider = make(map[string]Provider)
		for _, provider := range m.providers {
			injectedProviders[genProviderKey(provider)] = provider
		}

		// sort injected providers at head of provider list
		// to make it run NewProvider first
		for _, provider := range m.providers {
			componentType := reflect.TypeOf(provider)

			for j := 0; j < componentType.NumField(); j++ {
				componentField := componentType.Field(j)
				componentFieldType := componentField.Type
				componentFieldKey := genFieldKey(componentFieldType)

				if injectedProviders[componentFieldKey] != nil {
					m.providers = append([]Provider{injectedProviders[componentFieldKey]}, m.providers...)
				}
			}
		}

		// inject providers into providers
		for i, provider := range m.providers {
			newProvider, err := injectDependencies(provider, "provider", injectedProviders)
			if err != nil {
				panic(err)
			}

			providerKey := genProviderKey(provider)

			if providerInjectCheck[providerKey] == nil {
				providerInjectCheck[providerKey] = newProvider.Interface().(Provider).NewProvider()
			}

			m.providers[i] = providerInjectCheck[providerKey]
			injectedProviders[providerKey] = providerInjectCheck[providerKey]
		}

		// inject providers into controllers
		if utils.ArrIncludes(modulesInjectedFromMain, reflect.ValueOf(m).Pointer()) {
			for i, controller := range m.controllers {
				newController, err := injectDependencies(controller, "controller", injectedProviders)
				if err != nil {
					panic(err)
				}

				m.controllers[i] = newController.Interface().(Controller).NewController()

				// Handle REST
				if _, ok := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[0]); ok {
					rest := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[0]).Interface().(common.REST)
					controllerPath := reflect.TypeOf(m.controllers[i]).PkgPath()
					modulePrefixes := []string{}

					for _, globalPrefixes := range globalPrefixArr {
						for controllerKey, globalPrefixValues := range globalPrefixes {
							if getPkgFromControllerKey(controllerKey) == genFieldKey(reflect.TypeOf(controller)) {
								modulePrefixes = append(modulePrefixes, globalPrefixValues...)
							}
						}
					}

					for j := 0; j < reflect.TypeOf(m.controllers[i]).NumMethod(); j++ {
						methodName := reflect.TypeOf(m.controllers[i]).Method(j).Name

						// for main handler
						handler := reflect.ValueOf(m.controllers[i]).Method(j).Interface()
						rest.AddHandlerToRouterMap(modulePrefixes, methodName, handler)
					}

					// apply controller bound exception filers
					if _, loadedExceptionFilter := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[6]); loadedExceptionFilter {
						exceptionFilter := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[6]).Interface().(common.ExceptionFilter)

						exceptionFilterItemArr := exceptionFilter.
							InjectProvidersIntoRESTExceptionFilters(
								&rest,
								func(
									_ int,
									exceptionFilterableType reflect.Type,
									exceptionFilterableValue,
									newExceptionFilter reflect.Value,
								) {

									// callback use to inject providers
									// into exceptionFilter
									exceptionFilterField := exceptionFilterableType.Field(i)
									exceptionFilterFieldType := exceptionFilterField.Type
									exceptionFilterFieldNameKey := exceptionFilterField.Name
									injectProviderKey := exceptionFilterFieldType.PkgPath() + "/" + exceptionFilterFieldType.String()

									if !token.IsExported(exceptionFilterFieldNameKey) {
										panic(errors.New(
											utils.FmtRed(
												"can't set value to unexported '%v' field of the '%v' exceptionFilter",
												exceptionFilterFieldNameKey,
												exceptionFilterableType.Name(),
											),
										))
									}

									// Inject providers into exceptionFilter
									// inject provider priorities
									// local inject
									// global inject
									// inner packages
									// resolve dependencies error
									if injectedProviders[injectProviderKey] != nil {
										newExceptionFilter.Elem().Field(i).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
									} else if globalProviders[injectProviderKey] != nil {
										newExceptionFilter.Elem().Field(i).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
									} else if globalInterfaces[injectProviderKey] != nil {
										newExceptionFilter.Elem().Field(i).Set(reflect.ValueOf(globalInterfaces[injectProviderKey]))
									} else if !isInjectedProvider(exceptionFilterFieldType) {
										newExceptionFilter.Elem().Field(i).Set(exceptionFilterableValue.Field(i))
									} else {
										panic(errors.New(
											utils.FmtRed(
												"can't resolve dependencies of the '%v' provider. Please make sure that the argument dependency at index [%v] is available in the '%v' exceptionFilter",
												exceptionFilterFieldType.String(),
												i,
												exceptionFilterableType.Name(),
											),
										))
									}
								})

						// apply controller bound exceptionFilters
						for _, exceptionFilterItem := range exceptionFilterItemArr {
							m.RESTExceptionFilters = append(m.RESTExceptionFilters, common.RESTLayer{
								ControllerPath:  controllerPath,
								Method:          exceptionFilterItem.REST.Method,
								Route:           exceptionFilterItem.REST.Route,
								Version:         exceptionFilterItem.REST.Version,
								Handler:         exceptionFilterItem.REST.Common.Handler,
								Name:            exceptionFilterItem.REST.Common.Name,
								MainHandlerName: exceptionFilterItem.REST.Common.MainHandlerName,
								Pattern:         exceptionFilterItem.REST.Pattern,
							})
						}
					}

					// apply controller bound middleware
					if _, loadedMiddleware := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[10]); loadedMiddleware {
						middleware := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[10]).Interface().(common.Middleware)
						middlewareItemArr := middleware.InjectProvidersIntoRESTMiddlewares(&rest, func(i int, middlewareFnType reflect.Type, middlewareFnValue, newMiddleware reflect.Value) {

							// callback use to inject providers
							// into middleware
							middlewareField := middlewareFnType.Field(i)
							middlewareFieldType := middlewareField.Type
							middlewareFieldNameKey := middlewareField.Name
							injectProviderKey := middlewareFieldType.PkgPath() + "/" + middlewareFieldType.String()

							if !token.IsExported(middlewareFieldNameKey) {
								panic(errors.New(
									utils.FmtRed(
										"can't set value to unexported '%v' field of the '%v' middleware function",
										middlewareFieldNameKey,
										middlewareFnType.Name(),
									),
								))
							}

							// Inject providers into middleware
							// inject provider priorities
							// local inject
							// global inject
							// inner packages
							// resolve dependencies error
							if injectedProviders[injectProviderKey] != nil {
								newMiddleware.Elem().Field(i).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
							} else if globalProviders[injectProviderKey] != nil {
								newMiddleware.Elem().Field(i).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
							} else if globalInterfaces[injectProviderKey] != nil {
								newMiddleware.Elem().Field(i).Set(reflect.ValueOf(globalInterfaces[injectProviderKey]))
							} else if !isInjectedProvider(middlewareFieldType) {
								newMiddleware.Elem().Field(i).Set(middlewareFnValue.Field(i))
							} else {
								panic(errors.New(
									utils.FmtRed(
										"can't resolve dependencies of the '%v' provider. Please make sure that the argument dependency at index [%v] is available in the '%v' middleware function",
										middlewareFieldType.String(),
										i,
										middlewareFnType.Name(),
									),
								))
							}
						})

						// apply controller bound middlewares
						for _, middlewareItem := range middlewareItemArr {
							m.RESTMiddlewares = append(m.RESTMiddlewares, common.RESTLayer{
								ControllerPath:  controllerPath,
								Method:          middlewareItem.REST.Method,
								Route:           middlewareItem.REST.Route,
								Version:         middlewareItem.REST.Version,
								Handler:         middlewareItem.REST.Common.Handler,
								Name:            middlewareItem.REST.Common.Name,
								MainHandlerName: middlewareItem.REST.Common.MainHandlerName,
								Pattern:         middlewareItem.REST.Pattern,
							})
						}
					}

					// apply controller bound guard
					if _, loadedGuard := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[2]); loadedGuard {
						guard := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[2]).Interface().(common.Guard)
						guardItemArr := guard.InjectProvidersIntoRESTGuards(&rest, func(i int, guarderType reflect.Type, guarderValue, newGuard reflect.Value) {

							// callback use to inject providers
							// into guard
							guardField := guarderType.Field(i)
							guardFieldType := guardField.Type
							guardFieldNameKey := guardField.Name
							injectProviderKey := guardFieldType.PkgPath() + "/" + guardFieldType.String()

							if !token.IsExported(guardFieldNameKey) {
								panic(errors.New(
									utils.FmtRed(
										"can't set value to unexported '%v' field of the '%v' guarder",
										guardFieldNameKey,
										guarderType.Name(),
									),
								))
							}

							// Inject providers into guard
							// inject provider priorities
							// local inject
							// global inject
							// inner packages
							// resolve dependencies error
							if injectedProviders[injectProviderKey] != nil {
								newGuard.Elem().Field(i).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
							} else if globalProviders[injectProviderKey] != nil {
								newGuard.Elem().Field(i).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
							} else if globalInterfaces[injectProviderKey] != nil {
								newGuard.Elem().Field(i).Set(reflect.ValueOf(globalInterfaces[injectProviderKey]))
							} else if !isInjectedProvider(guardFieldType) {
								newGuard.Elem().Field(i).Set(guarderValue.Field(i))
							} else {
								panic(errors.New(
									utils.FmtRed(
										"can't resolve dependencies of the '%v' provider. Please make sure that the argument dependency at index [%v] is available in the '%v' guarder",
										guardFieldType.String(),
										i,
										guarderType.Name(),
									),
								))
							}
						})

						// apply controller bound guards
						for _, guardItem := range guardItemArr {
							m.RESTGuards = append(m.RESTGuards, common.RESTLayer{
								ControllerPath:  controllerPath,
								Method:          guardItem.REST.Method,
								Route:           guardItem.REST.Route,
								Version:         guardItem.REST.Version,
								Handler:         guardItem.REST.Common.Handler,
								Name:            guardItem.REST.Common.Name,
								MainHandlerName: guardItem.REST.Common.MainHandlerName,
								Pattern:         guardItem.REST.Pattern,
							})
						}
					}

					// apply controller bound interceptor
					if _, loadedInterceptor := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[4]); loadedInterceptor {
						interceptor := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[4]).Interface().(common.Interceptor)
						interceptorItemArr := interceptor.InjectProvidersIntoRESTInterceptors(&rest, func(i int, interceptableType reflect.Type, interceptableValue, newInterceptor reflect.Value) {

							// callback use to inject providers
							// into interceptor
							interceptorField := interceptableType.Field(i)
							interceptorFieldType := interceptorField.Type
							interceptorFieldNameKey := interceptorField.Name
							injectProviderKey := interceptorFieldType.PkgPath() + "/" + interceptorFieldType.String()

							if !token.IsExported(interceptorFieldNameKey) {
								panic(errors.New(
									utils.FmtRed(
										"can't set value to unexported '%v' field of the '%v' interceptor",
										interceptorFieldNameKey,
										interceptableType.Name(),
									),
								))
							}

							// Inject providers into interceptor
							// inject provider priorities
							// local inject
							// global inject
							// inner packages
							// resolve dependencies error
							if injectedProviders[injectProviderKey] != nil {
								newInterceptor.Elem().Field(i).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
							} else if globalProviders[injectProviderKey] != nil {
								newInterceptor.Elem().Field(i).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
							} else if globalInterfaces[injectProviderKey] != nil {
								newInterceptor.Elem().Field(i).Set(reflect.ValueOf(globalInterfaces[injectProviderKey]))
							} else if !isInjectedProvider(interceptorFieldType) {
								newInterceptor.Elem().Field(i).Set(interceptableValue.Field(i))
							} else {
								panic(errors.New(
									utils.FmtRed(
										"can't resolve dependencies of the '%v' provider. Please make sure that the argument dependency at index [%v] is available in the '%v' interceptor",
										interceptorFieldType.String(),
										i,
										interceptableType.Name(),
									),
								))
							}
						})

						// apply controller bound interceptors
						for _, interceptorItem := range interceptorItemArr {
							m.RESTInterceptors = append(m.RESTInterceptors, common.RESTLayer{
								ControllerPath:  controllerPath,
								Method:          interceptorItem.REST.Method,
								Route:           interceptorItem.REST.Route,
								Version:         interceptorItem.REST.Version,
								Handler:         interceptorItem.REST.Common.Handler,
								Name:            interceptorItem.REST.Common.Name,
								MainHandlerName: interceptorItem.REST.Common.MainHandlerName,
								Pattern:         interceptorItem.REST.Pattern,
							})
						}
					}

					// add main handler
					// for mainhandler: name = mainHandlerName
					// add for consistency with another layers
					for pattern, handler := range rest.RouterMap {
						if err := isInjectableHandler(handler, injectedProviders); err != nil {
							panic(utils.FmtRed("%s", err.Error()))
						}
						method, route, version := routing.PatternToMethodRouteVersion(pattern)
						m.RESTMainHandlers = append(m.RESTMainHandlers, common.RESTLayer{
							ControllerPath:  controllerPath,
							Method:          method,
							Route:           routing.ToEndpoint(route),
							Version:         version,
							Handler:         handler,
							Name:            rest.PatternToFnNameMap[pattern],
							MainHandlerName: rest.PatternToFnNameMap[pattern],
							Pattern:         pattern,
						})
					}
				}

				// Handle WS
				if _, ok := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[8]); ok {
					ws := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[8]).Interface().(common.WS)
					// controllerName := reflect.TypeOf(m.controllers[i]).PkgPath()

					for j := 0; j < reflect.TypeOf(m.controllers[i]).NumMethod(); j++ {
						methodName := reflect.TypeOf(m.controllers[i]).Method(j).Name

						// for main handler
						handler := reflect.ValueOf(m.controllers[i]).Method(j).Interface()
						ws.AddHandlerToEventMap(ws.GetSubprotocol(), methodName, handler)
					}

					// apply module bound middlewares
					// TODO: handle later

					// apply controller bound guard
					if _, loadedGuard := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[2]); loadedGuard {
						guard := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[2]).Interface().(common.Guard)
						guardItemArr := guard.InjectProvidersIntoWSGuards(&ws, func(i int, guarderType reflect.Type, guarderValue, newGuard reflect.Value) {

							// callback use to inject providers
							// into guard
							guardField := guarderType.Field(i)
							guardFieldType := guardField.Type
							guardFieldNameKey := guardField.Name
							injectProviderKey := guardFieldType.PkgPath() + "/" + guardFieldType.String()

							if !token.IsExported(guardFieldNameKey) {
								panic(errors.New(
									utils.FmtRed(
										"can't set value to unexported '%v' field of the '%v' guarder",
										guardFieldNameKey,
										guarderType.Name(),
									),
								))
							}

							// Inject providers into guard
							// inject provider priorities
							// local inject
							// global inject
							// inner packages
							// resolve dependencies error
							if injectedProviders[injectProviderKey] != nil {
								newGuard.Elem().Field(i).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
							} else if globalProviders[injectProviderKey] != nil {
								newGuard.Elem().Field(i).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
							} else if globalInterfaces[injectProviderKey] != nil {
								newGuard.Elem().Field(i).Set(reflect.ValueOf(globalInterfaces[injectProviderKey]))
							} else if !isInjectedProvider(guardFieldType) {
								newGuard.Elem().Field(i).Set(guarderValue.Field(i))
							} else {
								panic(errors.New(
									utils.FmtRed(
										"can't resolve dependencies of the '%v' provider. Please make sure that the argument dependency at index [%v] is available in the '%v' guarder",
										guardFieldType.String(),
										i,
										guarderType.Name(),
									),
								))
							}
						})

						// apply controller bound guards
						for _, guardItem := range guardItemArr {
							m.WSGuards = append(m.WSGuards, struct {
								Subprotocol string
								EventName   string
								Handler     any
							}{
								Subprotocol: ws.GetSubprotocol(),
								EventName:   guardItem.WS.EventName,
								Handler:     guardItem.WS.Common.Handler,
							})
						}
					}

					// apply controller bound interceptor
					if _, loadedInterceptor := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[4]); loadedInterceptor {
						interceptor := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[4]).Interface().(common.Interceptor)
						interceptorItemArr := interceptor.InjectProvidersIntoWSInterceptors(&ws, func(i int, interceptableType reflect.Type, interceptableValue, newInterceptor reflect.Value) {

							// callback use to inject providers
							// into interceptor
							interceptorField := interceptableType.Field(i)
							interceptorFieldType := interceptorField.Type
							interceptorFieldNameKey := interceptorField.Name
							injectProviderKey := interceptorFieldType.PkgPath() + "/" + interceptorFieldType.String()

							if !token.IsExported(interceptorFieldNameKey) {
								panic(errors.New(
									utils.FmtRed(
										"can't set value to unexported '%v' field of the '%v' interceptor",
										interceptorFieldNameKey,
										interceptableType.Name(),
									),
								))
							}

							// Inject providers into interceptor
							// inject provider priorities
							// local inject
							// global inject
							// inner packages
							// resolve dependencies error
							if injectedProviders[injectProviderKey] != nil {
								newInterceptor.Elem().Field(i).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
							} else if globalProviders[injectProviderKey] != nil {
								newInterceptor.Elem().Field(i).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
							} else if globalInterfaces[injectProviderKey] != nil {
								newInterceptor.Elem().Field(i).Set(reflect.ValueOf(globalInterfaces[injectProviderKey]))
							} else if !isInjectedProvider(interceptorFieldType) {
								newInterceptor.Elem().Field(i).Set(interceptableValue.Field(i))
							} else {
								panic(errors.New(
									utils.FmtRed(
										"can't resolve dependencies of the '%v' provider. Please make sure that the argument dependency at index [%v] is available in the '%v' interceptor",
										interceptorFieldType.String(),
										i,
										interceptableType.Name(),
									),
								))
							}
						})

						// apply controller bound interceptors
						for _, interceptorItem := range interceptorItemArr {
							m.WSInterceptors = append(m.WSInterceptors, struct {
								Subprotocol string
								EventName   string
								Handler     any
							}{
								Subprotocol: ws.GetSubprotocol(),
								EventName:   interceptorItem.WS.EventName,
								Handler:     interceptorItem.WS.Common.Handler,
							})
						}
					}

					// apply controller bound exception filer
					if _, loadedExceptionFilter := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[6]); loadedExceptionFilter {
						exceptionFilter := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[6]).Interface().(common.ExceptionFilter)
						exceptionFilterItemArr := exceptionFilter.InjectProvidersIntoWSExceptionFilters(&ws, func(i int, exceptionFilterableType reflect.Type, exceptionFilterableValue, newExceptionFilter reflect.Value) {

							// callback use to inject providers
							// into exceptionFilter
							exceptionFilterField := exceptionFilterableType.Field(i)
							exceptionFilterFieldType := exceptionFilterField.Type
							exceptionFilterFieldNameKey := exceptionFilterField.Name
							injectProviderKey := exceptionFilterFieldType.PkgPath() + "/" + exceptionFilterFieldType.String()

							if !token.IsExported(exceptionFilterFieldNameKey) {
								panic(errors.New(
									utils.FmtRed(
										"can't set value to unexported '%v' field of the '%v' exceptionFilter",
										exceptionFilterFieldNameKey,
										exceptionFilterableType.Name(),
									),
								))
							}

							// Inject providers into exceptionFilter
							// inject provider priorities
							// local inject
							// global inject
							// inner packages
							// resolve dependencies error
							if injectedProviders[injectProviderKey] != nil {
								newExceptionFilter.Elem().Field(i).Set(reflect.ValueOf(injectedProviders[injectProviderKey]))
							} else if globalProviders[injectProviderKey] != nil {
								newExceptionFilter.Elem().Field(i).Set(reflect.ValueOf(globalProviders[injectProviderKey]))
							} else if globalInterfaces[injectProviderKey] != nil {
								newExceptionFilter.Elem().Field(i).Set(reflect.ValueOf(globalInterfaces[injectProviderKey]))
							} else if !isInjectedProvider(exceptionFilterFieldType) {
								newExceptionFilter.Elem().Field(i).Set(exceptionFilterableValue.Field(i))
							} else {
								panic(errors.New(
									utils.FmtRed(
										"can't resolve dependencies of the '%v' provider. Please make sure that the argument dependency at index [%v] is available in the '%v' exceptionFilter",
										exceptionFilterFieldType.String(),
										i,
										exceptionFilterableType.Name(),
									),
								))
							}
						})

						// apply controller bound exceptionFilters
						for _, exceptionFilterItem := range exceptionFilterItemArr {
							m.WSExceptionFilters = append(m.WSExceptionFilters, struct {
								Subprotocol string
								EventName   string
								Handler     any
							}{
								Subprotocol: ws.GetSubprotocol(),
								EventName:   exceptionFilterItem.WS.EventName,
								Handler:     exceptionFilterItem.WS.Common.Handler,
							})
						}
					}

					// add ws main handler
					for eventName, handler := range ws.EventMap {

						if err := isInjectableHandler(handler, injectedProviders); err != nil {
							panic(utils.FmtRed("%s", err.Error()))
						}

						m.WSMainHandlers = append(m.WSMainHandlers, struct {
							Subprotocol string
							EventName   string
							Handler     any
						}{
							Subprotocol: ws.GetSubprotocol(),
							EventName:   eventName,
							Handler:     handler,
						})
					}
				}
			}
		}
	}

	return m
}
