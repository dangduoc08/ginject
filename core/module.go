package core

import (
	"reflect"
	"slices"
	"sync"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/routing"
)

var moduleGlobalMu sync.Mutex
var mainModulePtr uintptr
var modulesInjectedFromMain []uintptr
var staticModuleByDynamicPtr = make(map[uintptr]*Module)
var globalPrefixesByController = make(map[string][]string)

// globalProviderByKey and globalInterfaceByKey are read on every request
// that resolves a pipeable handler parameter (see injectDependencies,
// reached from getFnArgsByType), concurrently with other goroutines
// potentially still calling App.Create/UseLogger. sync.Map gives them their
// own internal synchronization instead of relying on moduleGlobalMu, which
// would otherwise have to be taken on the hot request path.
var globalProviderByKey sync.Map  // map[string]Provider
var globalInterfaceByKey sync.Map // map[string]any
var providerSingletonByKey map[string]Provider = make(map[string]Provider)
var fieldNameByRole = map[string]string{
	"rest":            "REST",
	"guard":           "Guard",
	"interceptor":     "Interceptor",
	"exceptionFilter": "ExceptionFilter",
	"ws":              "WS",
	"middleware":      "Middleware",
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

	// store WS module guards
	WSGuards []common.WSLayer

	// store WS module interceptors
	WSInterceptors []common.WSLayer

	// store WS module exception filters
	WSExceptionFilters []common.WSLayer

	// store WS main handlers
	WSMainHandlers []common.WSLayer
}

func (m *Module) injectGlobalProviders() {
	for _, provider := range m.providers {

		// generate a unique key for the provider
		globalProviderByKey.Store(genProviderKey(provider), provider)
	}
}

func (m *Module) Prefix(prefix string) *Module {
	m.prefixes = append([]string{str.Enclose(prefix, '/')}, m.prefixes...)

	return m
}

func (m *Module) ID() string {
	return m.id
}

func (m *Module) NewModule() *Module {
	m.Lock()
	defer m.Unlock()

	if m.singleInstance != nil {
		return m
	}
	m.singleInstance = m

	if m.OnInit != nil {
		m.OnInit()
	}

	m.bootstrapMainModule()
	m.injectStaticModules()
	m.injectDynamicModules()
	m.registerControllerPrefixes()

	injectedProviders := m.injectProviders()

	// only modules injected by main module are able to use controllers
	if slices.Contains(modulesInjectedFromMain, reflect.ValueOf(m).Pointer()) {
		m.injectControllers(injectedProviders)
	}

	return m
}

func (m *Module) bootstrapMainModule() {
	moduleGlobalMu.Lock()
	defer moduleGlobalMu.Unlock()

	if mainModulePtr != 0 {
		return
	}

	modulesInjectedFromMain = append(modulesInjectedFromMain, reflect.ValueOf(m).Pointer())
	mainModulePtr = reflect.ValueOf(m).Pointer()

	// main module's provider always injects globally
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
		staticModuleByDynamicPtr[reflect.ValueOf(dynamicModule).Pointer()] = staticModule

		m.controllers = append(m.controllers, staticModule.controllers...)
		m.providers = append(m.providers, staticModule.providers...)

		// dynamic modules which set as globally
		// have to be injected in main module
		if staticModule.IsGlobal {
			staticModule.injectGlobalProviders()
		}
	}
}

func (m *Module) injectStaticModules() {
	for _, staticModule := range m.staticModules {

		// no need to inject global here since globally static modules
		// should already be injected from main to make them injectable

		injectModule := staticModule.NewModule()
		if len(injectModule.providers) > 0 {
			m.providers = append(injectModule.providers, m.providers...)
		}
		if len(injectModule.controllers) > 0 {
			m.controllers = append(injectModule.controllers, m.controllers...)
		}
		toUniqueControllers(m, &m.controllers)
	}
}

func (m *Module) injectDynamicModules() {
	for _, dynamicModule := range m.dynamicModules {
		var staticModule *Module

		dynamicModulePtr := reflect.ValueOf(dynamicModule).Pointer()

		moduleGlobalMu.Lock()
		if storedInjectModule, ok := staticModuleByDynamicPtr[dynamicModulePtr]; ok {
			staticModule = storedInjectModule
		} else {
			staticModule = createStaticModuleFromDynamicModule(dynamicModule)
			staticModuleByDynamicPtr[dynamicModulePtr] = staticModule
		}
		moduleGlobalMu.Unlock()

		injectModule := staticModule.NewModule()
		if len(injectModule.providers) > 0 {
			m.providers = append(injectModule.providers, m.providers...)
		}
		if len(injectModule.controllers) > 0 {
			m.controllers = append(injectModule.controllers, m.controllers...)
		}
		toUniqueControllers(m, &m.controllers)
	}
}

func (m *Module) registerControllerPrefixes() {
	moduleGlobalMu.Lock()
	defer moduleGlobalMu.Unlock()

	for _, controller := range m.controllers {
		key := genFieldKey(reflect.TypeOf(controller))
		globalPrefixesByController[key] = append(globalPrefixesByController[key], m.prefixes...)
	}
}

func (m *Module) injectProviders() map[string]Provider {
	injectedProviders := make(map[string]Provider)
	for _, provider := range m.providers {
		injectedProviders[genProviderKey(provider)] = provider
	}

	// sort injected providers at head of provider list
	// to make it run NewProvider first
	for _, provider := range m.providers {
		componentType := reflect.TypeOf(provider)

		for j := 0; j < componentType.NumField(); j++ {
			componentFieldKey := genFieldKey(componentType.Field(j).Type)

			if injectedProviders[componentFieldKey] != nil {
				m.providers = append([]Provider{injectedProviders[componentFieldKey]}, m.providers...)
			}
		}
	}

	// inject providers into providers
	moduleGlobalMu.Lock()
	for i, provider := range m.providers {
		newProvider, err := injectDependencies(provider, "provider", injectedProviders)
		if err != nil {
			moduleGlobalMu.Unlock()
			panic(err)
		}

		providerKey := genProviderKey(provider)

		if providerSingletonByKey[providerKey] == nil {
			providerSingletonByKey[providerKey] = newProvider.Interface().(Provider).NewProvider()
		}

		m.providers[i] = providerSingletonByKey[providerKey]
		injectedProviders[providerKey] = providerSingletonByKey[providerKey]
	}
	moduleGlobalMu.Unlock()

	return injectedProviders
}

func (m *Module) injectControllers(injectedProviders map[string]Provider) {
	moduleGlobalMu.Lock()
	defer moduleGlobalMu.Unlock()

	for i, controller := range m.controllers {
		newController, err := injectDependencies(controller, "controller", injectedProviders)
		if err != nil {
			panic(err)
		}

		m.controllers[i] = newController.Interface().(Controller).NewController()

		m.bindRESTController(m.controllers[i], injectedProviders)
		m.bindWSController(m.controllers[i], injectedProviders)
	}
}

func controllerModulePrefixes(controllerType reflect.Type) []string {
	return globalPrefixesByController[genFieldKey(controllerType)]
}

func (m *Module) bindRESTController(controller Controller, injectedProviders map[string]Provider) {
	controllerType := reflect.TypeOf(controller)
	if _, ok := controllerType.FieldByName(fieldNameByRole["rest"]); !ok {
		return
	}

	rest := reflect.ValueOf(controller).FieldByName(fieldNameByRole["rest"]).Interface().(common.REST)
	controllerPath := controllerType.PkgPath()
	modulePrefixes := controllerModulePrefixes(controllerType)

	for j := 0; j < controllerType.NumMethod(); j++ {
		methodName := controllerType.Method(j).Name

		// for main handler
		handler := reflect.ValueOf(controller).Method(j).Interface()
		rest.AddHandlerToRouterMap(modulePrefixes, methodName, handler)
	}

	m.bindRESTExceptionFilters(controller, controllerType, controllerPath, &rest, injectedProviders)
	m.bindRESTMiddlewares(controller, controllerType, controllerPath, &rest, injectedProviders)
	m.bindRESTGuards(controller, controllerType, controllerPath, &rest, injectedProviders)
	m.bindRESTInterceptors(controller, controllerType, controllerPath, &rest, injectedProviders)

	// add main handler
	// for mainhandler: name = mainHandlerName
	// add for consistency with another layers
	for pattern, handler := range rest.RouterMap {
		if err := isInjectableHandler(handler, injectedProviders, knownRESTDependencyKeys); err != nil {
			panic(color.FmtRed("%s", err.Error()))
		}
		method, route, version := routing.PatternToMethodRouteVersion(pattern)
		m.RESTMainHandlers = append(m.RESTMainHandlers, common.RESTLayer{
			ControllerPath:  controllerPath,
			Method:          method,
			Route:           str.Enclose(route, '/'),
			Version:         version,
			Handler:         handler,
			Name:            rest.PatternToFuncNameMap[pattern],
			MainHandlerName: rest.PatternToFuncNameMap[pattern],
			Pattern:         pattern,
		})
	}
}

func (m *Module) bindRESTExceptionFilters(controller Controller, controllerType reflect.Type, controllerPath string, rest *common.REST, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["exceptionFilter"]); !ok {
		return
	}

	exceptionFilter := reflect.ValueOf(controller).FieldByName(fieldNameByRole["exceptionFilter"]).Interface().(common.ExceptionFilter)
	items := exceptionFilter.InjectProvidersIntoRESTExceptionFilters(rest, buildFieldInjectionCallback("exceptionFilter", injectedProviders))

	for _, item := range items {
		m.RESTExceptionFilters = append(m.RESTExceptionFilters, common.RESTLayer{
			ControllerPath:  controllerPath,
			Method:          item.REST.Method,
			Route:           item.REST.Route,
			Version:         item.REST.Version,
			Handler:         item.REST.Common.Handler,
			Name:            item.REST.Common.Name,
			MainHandlerName: item.REST.Common.MainHandlerName,
			Pattern:         item.REST.Pattern,
		})
	}
}

func (m *Module) bindRESTMiddlewares(controller Controller, controllerType reflect.Type, controllerPath string, rest *common.REST, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["middleware"]); !ok {
		return
	}

	middleware := reflect.ValueOf(controller).FieldByName(fieldNameByRole["middleware"]).Interface().(common.Middleware)
	items := middleware.InjectProvidersIntoRESTMiddlewares(rest, buildFieldInjectionCallback("middleware function", injectedProviders))

	for _, item := range items {
		m.RESTMiddlewares = append(m.RESTMiddlewares, common.RESTLayer{
			ControllerPath:  controllerPath,
			Method:          item.REST.Method,
			Route:           item.REST.Route,
			Version:         item.REST.Version,
			Handler:         item.REST.Common.Handler,
			Name:            item.REST.Common.Name,
			MainHandlerName: item.REST.Common.MainHandlerName,
			Pattern:         item.REST.Pattern,
		})
	}
}

func (m *Module) bindRESTGuards(controller Controller, controllerType reflect.Type, controllerPath string, rest *common.REST, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["guard"]); !ok {
		return
	}

	guard := reflect.ValueOf(controller).FieldByName(fieldNameByRole["guard"]).Interface().(common.Guard)
	items := guard.InjectProvidersIntoRESTGuards(rest, buildFieldInjectionCallback("guarder", injectedProviders))

	for _, item := range items {
		m.RESTGuards = append(m.RESTGuards, common.RESTLayer{
			ControllerPath:  controllerPath,
			Method:          item.REST.Method,
			Route:           item.REST.Route,
			Version:         item.REST.Version,
			Handler:         item.REST.Common.Handler,
			Name:            item.REST.Common.Name,
			MainHandlerName: item.REST.Common.MainHandlerName,
			Pattern:         item.REST.Pattern,
		})
	}
}

func (m *Module) bindRESTInterceptors(controller Controller, controllerType reflect.Type, controllerPath string, rest *common.REST, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["interceptor"]); !ok {
		return
	}

	interceptor := reflect.ValueOf(controller).FieldByName(fieldNameByRole["interceptor"]).Interface().(common.Interceptor)
	items := interceptor.InjectProvidersIntoRESTInterceptors(rest, buildFieldInjectionCallback("interceptor", injectedProviders))

	for _, item := range items {
		m.RESTInterceptors = append(m.RESTInterceptors, common.RESTLayer{
			ControllerPath:  controllerPath,
			Method:          item.REST.Method,
			Route:           item.REST.Route,
			Version:         item.REST.Version,
			Handler:         item.REST.Common.Handler,
			Name:            item.REST.Common.Name,
			MainHandlerName: item.REST.Common.MainHandlerName,
			Pattern:         item.REST.Pattern,
		})
	}
}

func (m *Module) bindWSController(controller Controller, injectedProviders map[string]Provider) {
	controllerType := reflect.TypeOf(controller)
	if _, ok := controllerType.FieldByName(fieldNameByRole["ws"]); !ok {
		return
	}

	ws := reflect.ValueOf(controller).FieldByName(fieldNameByRole["ws"]).Interface().(common.WS)

	for j := 0; j < controllerType.NumMethod(); j++ {
		methodName := controllerType.Method(j).Name

		// for main handler
		handler := reflect.ValueOf(controller).Method(j).Interface()
		ws.AddHandlerToEventMap(methodName, handler)
	}

	m.bindWSGuards(controller, controllerType, &ws, injectedProviders)
	m.bindWSInterceptors(controller, controllerType, &ws, injectedProviders)
	m.bindWSExceptionFilters(controller, controllerType, &ws, injectedProviders)

	// add ws main handler
	for eventName, handler := range ws.EventMap {
		if err := isInjectableHandler(handler, injectedProviders, knownWSDependencyKeys); err != nil {
			panic(color.FmtRed("%s", err.Error()))
		}

		m.WSMainHandlers = append(m.WSMainHandlers, common.WSLayer{
			EventName: eventName,
			Handler:   handler,
		})
	}
}

func (m *Module) bindWSGuards(controller Controller, controllerType reflect.Type, ws *common.WS, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["guard"]); !ok {
		return
	}

	guard := reflect.ValueOf(controller).FieldByName(fieldNameByRole["guard"]).Interface().(common.Guard)
	items := guard.InjectProvidersIntoWSGuards(ws, buildFieldInjectionCallback("guarder", injectedProviders))

	for _, item := range items {
		m.WSGuards = append(m.WSGuards, common.WSLayer{
			EventName: item.WS.EventName,
			Handler:   item.WS.Common.Handler,
		})
	}
}

func (m *Module) bindWSInterceptors(controller Controller, controllerType reflect.Type, ws *common.WS, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["interceptor"]); !ok {
		return
	}

	interceptor := reflect.ValueOf(controller).FieldByName(fieldNameByRole["interceptor"]).Interface().(common.Interceptor)
	items := interceptor.InjectProvidersIntoWSInterceptors(ws, buildFieldInjectionCallback("interceptor", injectedProviders))

	for _, item := range items {
		m.WSInterceptors = append(m.WSInterceptors, common.WSLayer{
			EventName: item.WS.EventName,
			Handler:   item.WS.Common.Handler,
		})
	}
}

func (m *Module) bindWSExceptionFilters(controller Controller, controllerType reflect.Type, ws *common.WS, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["exceptionFilter"]); !ok {
		return
	}

	exceptionFilter := reflect.ValueOf(controller).FieldByName(fieldNameByRole["exceptionFilter"]).Interface().(common.ExceptionFilter)
	items := exceptionFilter.InjectProvidersIntoWSExceptionFilters(ws, buildFieldInjectionCallback("exceptionFilter", injectedProviders))

	for _, item := range items {
		m.WSExceptionFilters = append(m.WSExceptionFilters, common.WSLayer{
			EventName: item.WS.EventName,
			Handler:   item.WS.Common.Handler,
		})
	}
}
