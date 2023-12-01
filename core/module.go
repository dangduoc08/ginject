package core

import (
	"fmt"
	"go/token"
	"reflect"
	"sync"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

var mainModule uintptr
var modulesInjectedFromMain []uintptr
var injectedDynamicModules []uintptr
var globalProviders map[string]Provider = make(map[string]Provider)
var globalInterfaces map[string]any = make(map[string]any)
var providerInjectCheck map[string]Provider = make(map[string]Provider)
var insertedRoutes = make(map[string]string)
var insertedEvents = make(map[string]string)
var noInjectedFields = []string{
	"Rest",
	"common.Rest",
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
	"github.com/dangduoc08/gooh/common/common.Logger",
}

type Module struct {
	*sync.Mutex
	singleInstance *Module
	staticModules  []*Module
	dynamicModules []any
	providers      []Provider
	exports        []Provider
	controllers    []Controller

	Middleware *Middleware
	IsGlobal   bool
	OnInit     func()

	// store REST module middlewares
	RESTMiddlewares []struct {
		Method   string
		Route    string
		Handlers []context.Handler
	}

	// store REST module guards
	RESTGuards []struct {
		Method  string
		Route   string
		Handler any
	}

	// store REST module interceptors
	RESTInterceptors []struct {
		Method  string
		Route   string
		Handler any
	}

	// store REST module exception filters
	RESTExceptionFilters []struct {
		Method  string
		Route   string
		Handler any
	}

	// store REST main handlers
	RESTMainHandlers []struct {
		Method  string
		Route   string
		Handler any
	}

	// store WS module middlewares
	WSMiddlewares []struct {
		Subprotocol string
		EventName   string
		Handlers    []context.Handler
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
		if mainModule == 0 {
			m.injectMainModules()
			mainModule = reflect.ValueOf(m).Pointer()

			// main module's provider
			// alway inject globally
			m.injectGlobalProviders()

			for _, staticModule := range m.staticModules {
				staticModule.injectMainModules()

				if staticModule.IsGlobal {
					staticModule.injectGlobalProviders()
				}
			}

			for _, dynamicModule := range m.dynamicModules {
				modulesInjectedFromMain = append(modulesInjectedFromMain, reflect.ValueOf(dynamicModule).Pointer())

				// dynamic module cannot be injected as globally
				// initially cannot resolve all dependencies
			}
		}

		// inject static modules
		for _, staticModule := range m.staticModules {

			// recursion injection
			injectModule := staticModule.NewModule()

			// only import providers which exported
			if len(injectModule.exports) > 0 {
				m.providers = append(m.providers, injectModule.exports...)
				m.exports = append(m.exports, injectModule.exports...)
			}

			if reflect.ValueOf(m).Pointer() == mainModule {
				m.RESTMiddlewares = append(m.RESTMiddlewares, injectModule.RESTMiddlewares...)
				m.RESTGuards = append(m.RESTGuards, injectModule.RESTGuards...)
				m.RESTInterceptors = append(m.RESTInterceptors, injectModule.RESTInterceptors...)
				m.RESTExceptionFilters = append(m.RESTExceptionFilters, injectModule.RESTExceptionFilters...)
				m.RESTMainHandlers = append(m.RESTMainHandlers, injectModule.RESTMainHandlers...)

				m.WSMiddlewares = append(m.WSMiddlewares, injectModule.WSMiddlewares...)
				m.WSGuards = append(m.WSGuards, injectModule.WSGuards...)
				m.WSInterceptors = append(m.WSInterceptors, injectModule.WSInterceptors...)
				m.WSExceptionFilters = append(m.WSExceptionFilters, injectModule.WSExceptionFilters...)
				m.WSMainHandlers = append(m.WSMainHandlers, injectModule.WSMainHandlers...)
			}
		}

		// inject local providers
		// from static modules
		var injectedProviders map[string]Provider = make(map[string]Provider)
		for _, provider := range m.providers {
			injectedProviders[genProviderKey(provider)] = provider
		}

		// inject providers into providers
		injectedErrorProviders := []Provider{}
		for i, provider := range m.providers {
			newProvider, err := injectDependencies(provider, "provider", injectedProviders)
			if err != nil {
				injectedErrorProviders = append(injectedErrorProviders, provider)
				continue
			}

			providerKey := genProviderKey(provider)

			if providerInjectCheck[providerKey] == nil {
				providerInjectCheck[providerKey] = newProvider.Interface().(Provider).NewProvider()
			}

			m.providers[i] = providerInjectCheck[providerKey]
			injectedProviders[providerKey] = providerInjectCheck[providerKey]
		}

		// inject dynamic modules
		for _, dynamicModule := range m.dynamicModules {
			dynamicModulePtr := reflect.ValueOf(dynamicModule).Pointer()
			if !utils.ArrIncludes(injectedDynamicModules, dynamicModulePtr) {
				injectedDynamicModules = append(injectedDynamicModules, dynamicModulePtr)
			}
			staticModule := createStaticModuleFromDynamicModule(dynamicModule, injectedProviders)

			// to replace dynamic module pointer
			// with new static module pointer
			// to make it able to create controller
			matchedIndex := utils.ArrFindIndex[uintptr](modulesInjectedFromMain, func(el uintptr, i int) bool {
				return el == reflect.ValueOf(dynamicModule).Pointer()
			})

			if matchedIndex > -1 {

				// re-assign controllers
				if len(staticModule.controllers) > 0 {
					m.controllers = staticModule.controllers
				}
				modulesInjectedFromMain[matchedIndex] = reflect.ValueOf(staticModule).Pointer()
			}

			if staticModule.IsGlobal {
				panic(fmt.Errorf(
					utils.FmtRed(
						"can't set dynamic module as global module",
					),
				))
			}

			injectModule := staticModule.NewModule()

			// only import providers which exported
			if len(injectModule.exports) > 0 {
				m.providers = append(m.providers, injectModule.exports...)
				m.exports = append(m.exports, injectModule.exports...)
			}

			if reflect.ValueOf(m).Pointer() == mainModule {
				m.RESTMiddlewares = append(m.RESTMiddlewares, injectModule.RESTMiddlewares...)
				m.RESTGuards = append(m.RESTGuards, injectModule.RESTGuards...)
				m.RESTInterceptors = append(m.RESTInterceptors, injectModule.RESTInterceptors...)
				m.RESTExceptionFilters = append(m.RESTExceptionFilters, injectModule.RESTExceptionFilters...)
				m.RESTMainHandlers = append(m.RESTMainHandlers, injectModule.RESTMainHandlers...)

				m.WSMiddlewares = append(m.WSMiddlewares, injectModule.WSMiddlewares...)
				m.WSGuards = append(m.WSGuards, injectModule.WSGuards...)
				m.WSInterceptors = append(m.WSInterceptors, injectModule.WSInterceptors...)
				m.WSExceptionFilters = append(m.WSExceptionFilters, injectModule.WSExceptionFilters...)
				m.WSMainHandlers = append(m.WSMainHandlers, injectModule.WSMainHandlers...)
			}
		}
		// inject local providers
		// from dynamic modules
		for _, provider := range m.providers {
			injectedProviders[genProviderKey(provider)] = provider
		}

		// inject error providers
		// this can happen
		// due to inject dynamic providers into static providers
		for i, provider := range injectedErrorProviders {
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
					rest := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[0]).Interface().(common.Rest)

					for j := 0; j < reflect.TypeOf(m.controllers[i]).NumMethod(); j++ {
						methodName := reflect.TypeOf(m.controllers[i]).Method(j).Name

						// for module middleware inclusion
						m.Middleware.includeREST(methodName)

						// for main handler
						handler := reflect.ValueOf(m.controllers[i]).Method(j).Interface()
						rest.AddHandlerToRouterMap(methodName, insertedRoutes, handler)
					}

					// create middlewareItemArr
					m.Middleware.addREST(rest.GetPrefixes())

					// apply module bound middlewares
					for _, middlewareItem := range m.Middleware.restMiddlewareItemArr {
						m.RESTMiddlewares = append(m.RESTMiddlewares, struct {
							Method   string
							Route    string
							Handlers []func(*context.Context)
						}{
							Method:   middlewareItem.method,
							Route:    middlewareItem.route,
							Handlers: middlewareItem.handlers,
						})
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
							m.RESTGuards = append(m.RESTGuards, struct {
								Method  string
								Route   string
								Handler any
							}{
								Method:  guardItem.Method,
								Route:   guardItem.Route,
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
							m.RESTInterceptors = append(m.RESTInterceptors, struct {
								Method  string
								Route   string
								Handler any
							}{
								Method:  interceptorItem.Method,
								Route:   interceptorItem.Route,
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
							m.RESTExceptionFilters = append(m.RESTExceptionFilters, struct {
								Method  string
								Route   string
								Handler any
							}{
								Method:  exceptionFilterItem.Method,
								Route:   exceptionFilterItem.Route,
								Handler: exceptionFilterItem.Handler,
							})
						}
					}

					// add main handler
					for pattern, handler := range rest.RouterMap {
						if err := isInjectableHandler(handler, injectedProviders); err != nil {
							panic(utils.FmtRed(err.Error()))
						}

						method, route := routing.SplitRoute(pattern)
						m.RESTMainHandlers = append(m.RESTMainHandlers, struct {
							Method  string
							Route   string
							Handler any
						}{
							Method:  method,
							Route:   routing.ToEndpoint(route),
							Handler: handler,
						})
					}
				}

				// Handle WS
				if _, ok := reflect.TypeOf(m.controllers[i]).FieldByName(noInjectedFields[8]); ok {
					ws := reflect.ValueOf(m.controllers[i]).FieldByName(noInjectedFields[8]).Interface().(common.WS)

					for j := 0; j < reflect.TypeOf(m.controllers[i]).NumMethod(); j++ {
						methodName := reflect.TypeOf(m.controllers[i]).Method(j).Name

						// for module middleware inclusion
						m.Middleware.includeWS(ws.GetSubprotocol(), methodName)

						// for main handler
						handler := reflect.ValueOf(m.controllers[i]).Method(j).Interface()
						ws.AddHandlerToEventMap(ws.GetSubprotocol(), methodName, insertedEvents, handler)
					}

					// create middlewareItemArr
					m.Middleware.addWS()

					// apply module bound middlewares
					for _, middlewareItem := range m.Middleware.wsMiddlewareItemArr {
						m.WSMiddlewares = append(m.WSMiddlewares, struct {
							Subprotocol string
							EventName   string
							Handlers    []func(*context.Context)
						}{
							Subprotocol: ws.GetSubprotocol(),
							EventName:   middlewareItem.eventName,
							Handlers:    middlewareItem.handlers,
						})
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
		}
	}

	return m
}
