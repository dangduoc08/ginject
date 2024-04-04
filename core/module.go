package core

import (
	"fmt"
	"go/token"
	"reflect"
	"sync"

	"github.com/dangduoc08/gogo/common"
	"github.com/dangduoc08/gogo/ctx"
	"github.com/dangduoc08/gogo/routing"
	"github.com/dangduoc08/gogo/utils"
)

var mainModulePtr uintptr
var modulesInjectedFromMain []uintptr
var injectedDynamicModules = make(map[uintptr]*Module)
var globalPrefixes = map[string][]string{}
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
}
var injectableInterfaces = []string{
	"github.com/dangduoc08/gogo/common/common.Logger",
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

	Middleware *Middleware
	IsGlobal   bool
	OnInit     func()

	// store REST module middlewares
	RESTMiddlewares []RESTMiddlewareLayer

	// store REST module guards
	RESTGuards []RESTCommonLayer

	// store REST module interceptors
	RESTInterceptors []RESTCommonLayer

	// store REST module exception filters
	RESTExceptionFilters []RESTCommonLayer

	// store REST main handlers
	RESTMainHandlers []RESTCommonLayer

	// store WS module middlewares
	WSMiddlewares []struct {
		controllerName string
		Subprotocol    string
		EventName      string
		Handlers       []ctx.Handler
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
			if len(injectModule.RESTMiddlewares) > 0 {
				m.RESTMiddlewares = append(injectModule.RESTMiddlewares, m.RESTMiddlewares...)
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
			if len(injectModule.RESTMiddlewares) > 0 {
				m.RESTMiddlewares = append(injectModule.RESTMiddlewares, m.RESTMiddlewares...)
			}
			toUniqueControllers(m, &m.controllers)
		}

		if len(m.Middleware.middlewares) > 0 {
			for _, controller := range m.controllers {
				controllerName := reflect.TypeOf(controller).PkgPath()
				m.Middleware.addREST(controllerName, &m.RESTMiddlewares)
				m.Middleware.addWS(controllerName, &m.WSMiddlewares)
			}
		}

		// set module prefixes
		for _, controller := range m.controllers {
			globalPrefixes[genControllerKey(m, controller)] = m.prefixes
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
			newRESTMiddlewares := []RESTMiddlewareLayer{}

			newWSMiddlewares := []struct {
				controllerName string
				Subprotocol    string
				EventName      string
				Handlers       []func(*ctx.Context)
			}{}

			for i, controller := range m.controllers {
				newController, err := injectDependencies(controller, "controller", injectedProviders)
				if err != nil {
					panic(err)
				}

				m.controllers[i] = newController.Interface().(Controller).NewController()

				// Handle REST
				if _, ok := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[0]); ok {
					rest := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[0]).Interface().(common.REST)
					controllerName := reflect.TypeOf(m.controllers[i]).PkgPath()
					modulePrefixes := []string{}

					for controllerKey, globalPrefixValues := range globalPrefixes {
						if getPkgFromControllerKey(controllerKey) == genFieldKey(reflect.TypeOf(controller)) {
							modulePrefixes = append(modulePrefixes, globalPrefixValues...)
						}
					}

					for j := 0; j < reflect.TypeOf(m.controllers[i]).NumMethod(); j++ {
						methodName := reflect.TypeOf(m.controllers[i]).Method(j).Name

						// for main handler
						handler := reflect.ValueOf(m.controllers[i]).Method(j).Interface()
						rest.AddHandlerToRouterMap(modulePrefixes, methodName, handler)
					}

					// apply controller bound middlewares
					for _, restModuleMiddleware := range m.RESTMiddlewares {
						if restModuleMiddleware.controllerName == controllerName {
							for pattern := range rest.RouterMap {
								if restModuleMiddleware.Method == "*" {
									method, route, version := routing.PatternToMethodRouteVersion(pattern)

									newRESTMiddlewares = append(
										newRESTMiddlewares,
										RESTMiddlewareLayer{
											controllerName: controllerName,
											Route:          route,
											Version:        version,
											Method:         method,
											Handlers:       restModuleMiddleware.Handlers,
										},
									)
								} else if rest.PatternToFnNameMap[pattern] == restModuleMiddleware.Route {
									method, route, version := routing.PatternToMethodRouteVersion(pattern)

									newRESTMiddlewares = append(
										newRESTMiddlewares,
										RESTMiddlewareLayer{
											controllerName: controllerName,
											Route:          route,
											Version:        version,
											Method:         method,
											Handlers:       restModuleMiddleware.Handlers,
										},
									)
								}
							}
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
								panic(fmt.Errorf(
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
								panic(fmt.Errorf(
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
							m.RESTGuards = append(m.RESTGuards, RESTCommonLayer{
								Method:  guardItem.Method,
								Route:   guardItem.Route,
								Version: guardItem.Version,
								Handler: guardItem.Handler,
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
								panic(fmt.Errorf(
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
								panic(fmt.Errorf(
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
							m.RESTInterceptors = append(m.RESTInterceptors, RESTCommonLayer{
								Method:  interceptorItem.Method,
								Route:   interceptorItem.Route,
								Version: interceptorItem.Version,
								Handler: interceptorItem.Handler,
							})
						}
					}

					// apply controller bound exception filer
					if _, loadedExceptionFilter := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[6]); loadedExceptionFilter {
						exceptionFilter := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[6]).Interface().(common.ExceptionFilter)
						exceptionFilterItemArr := exceptionFilter.InjectProvidersIntoRESTExceptionFilters(&rest, func(i int, exceptionFilterableType reflect.Type, exceptionFilterableValue, newExceptionFilter reflect.Value) {

							// callback use to inject providers
							// into exceptionFilter
							exceptionFilterField := exceptionFilterableType.Field(i)
							exceptionFilterFieldType := exceptionFilterField.Type
							exceptionFilterFieldNameKey := exceptionFilterField.Name
							injectProviderKey := exceptionFilterFieldType.PkgPath() + "/" + exceptionFilterFieldType.String()

							if !token.IsExported(exceptionFilterFieldNameKey) {
								panic(fmt.Errorf(
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
								panic(fmt.Errorf(
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
							m.RESTExceptionFilters = append(m.RESTExceptionFilters, RESTCommonLayer{
								Method:  exceptionFilterItem.Method,
								Route:   exceptionFilterItem.Route,
								Version: exceptionFilterItem.Version,
								Handler: exceptionFilterItem.Handler,
							})
						}
					}

					// add main handler
					for pattern, handler := range rest.RouterMap {
						if err := isInjectableHandler(handler, injectedProviders); err != nil {
							panic(utils.FmtRed(err.Error()))
						}

						method, route, version := routing.PatternToMethodRouteVersion(pattern)
						m.RESTMainHandlers = append(m.RESTMainHandlers, RESTCommonLayer{
							Method:  method,
							Route:   routing.ToEndpoint(route),
							Version: version,
							Handler: handler,
						})
					}
				}

				// Handle WS
				if _, ok := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[8]); ok {
					ws := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[8]).Interface().(common.WS)
					controllerName := reflect.TypeOf(m.controllers[i]).PkgPath()

					for j := 0; j < reflect.TypeOf(m.controllers[i]).NumMethod(); j++ {
						methodName := reflect.TypeOf(m.controllers[i]).Method(j).Name

						// for main handler
						handler := reflect.ValueOf(m.controllers[i]).Method(j).Interface()
						ws.AddHandlerToEventMap(ws.GetSubprotocol(), methodName, handler)
					}

					// apply controller bound middlewares
					for _, wsModuleMiddleware := range m.WSMiddlewares {
						if wsModuleMiddleware.controllerName == controllerName {
							for pattern := range ws.EventMap {

								if wsModuleMiddleware.Subprotocol == "*" {
									newWSMiddlewares = append(
										newWSMiddlewares,
										struct {
											controllerName string
											Subprotocol    string
											EventName      string
											Handlers       []func(*ctx.Context)
										}{
											controllerName: controllerName,
											Subprotocol:    ws.GetSubprotocol(),
											EventName:      pattern,
											Handlers:       wsModuleMiddleware.Handlers,
										},
									)
								} else if pattern == common.ToWSEventName(ws.GetSubprotocol(), wsModuleMiddleware.EventName) {
									newWSMiddlewares = append(
										newWSMiddlewares,
										struct {
											controllerName string
											Subprotocol    string
											EventName      string
											Handlers       []func(*ctx.Context)
										}{
											controllerName: controllerName,
											Subprotocol:    ws.GetSubprotocol(),
											EventName:      pattern,
											Handlers:       wsModuleMiddleware.Handlers,
										},
									)
								}
							}
						}
					}

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
								panic(fmt.Errorf(
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
								panic(fmt.Errorf(
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
								EventName:   guardItem.EventName,
								Handler:     guardItem.Handler,
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
								panic(fmt.Errorf(
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
								panic(fmt.Errorf(
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
								EventName:   interceptorItem.EventName,
								Handler:     interceptorItem.Handler,
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
								panic(fmt.Errorf(
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
								panic(fmt.Errorf(
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
								EventName:   exceptionFilterItem.EventName,
								Handler:     exceptionFilterItem.Handler,
							})
						}
					}

					// add ws main handler
					for eventName, handler := range ws.EventMap {

						if err := isInjectableHandler(handler, injectedProviders); err != nil {
							panic(utils.FmtRed(err.Error()))
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

			m.RESTMiddlewares = newRESTMiddlewares
			m.WSMiddlewares = newWSMiddlewares
		}
	}

	return m
}
