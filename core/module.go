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
	"http":            "HTTP",
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

	// store HTTP module exception filters
	HTTPExceptionFilters []common.HTTPLayer

	// store HTTP module middlewares
	HTTPMiddlewares []common.HTTPLayer

	// store HTTP module guards
	HTTPGuards []common.HTTPLayer

	// store HTTP module interceptors
	HTTPInterceptors []common.HTTPLayer

	// store HTTP main handlers
	HTTPMainHandlers []common.HTTPLayer

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

		m.bindHTTPController(m.controllers[i], injectedProviders)
		m.bindWSController(m.controllers[i], injectedProviders)
	}
}

func controllerModulePrefixes(controllerType reflect.Type) []string {
	return globalPrefixesByController[genFieldKey(controllerType)]
}

func (m *Module) bindHTTPController(controller Controller, injectedProviders map[string]Provider) {
	controllerType := reflect.TypeOf(controller)
	if _, ok := controllerType.FieldByName(fieldNameByRole["http"]); !ok {
		return
	}

	http := reflect.ValueOf(controller).FieldByName(fieldNameByRole["http"]).Interface().(common.HTTP)
	controllerPath := controllerType.PkgPath()
	modulePrefixes := controllerModulePrefixes(controllerType)

	for j := 0; j < controllerType.NumMethod(); j++ {
		methodName := controllerType.Method(j).Name

		// for main handler
		handler := reflect.ValueOf(controller).Method(j).Interface()
		http.AddHandlerToRouterMap(modulePrefixes, methodName, handler)
	}

	m.bindHTTPExceptionFilters(controller, controllerType, controllerPath, &http, injectedProviders)
	m.bindHTTPMiddlewares(controller, controllerType, controllerPath, &http, injectedProviders)
	m.bindHTTPGuards(controller, controllerType, controllerPath, &http, injectedProviders)
	m.bindHTTPInterceptors(controller, controllerType, controllerPath, &http, injectedProviders)

	// add main handler
	// for mainhandler: name = mainHandlerName
	// add for consistency with another layers
	for pattern, handler := range http.RouterMap {
		if err := isInjectableHandler(handler, injectedProviders, knownHTTPDependencyKeys); err != nil {
			panic(color.FmtRed("%s", err.Error()))
		}
		method, route, version := routing.PatternToMethodRouteVersion(pattern)
		m.HTTPMainHandlers = append(m.HTTPMainHandlers, common.HTTPLayer{
			ControllerPath:  controllerPath,
			Method:          method,
			Route:           str.Enclose(route, '/'),
			Version:         version,
			Handler:         handler,
			Name:            http.PatternToFuncNameMap[pattern],
			MainHandlerName: http.PatternToFuncNameMap[pattern],
			Pattern:         pattern,
		})
	}
}

func (m *Module) bindHTTPExceptionFilters(controller Controller, controllerType reflect.Type, controllerPath string, http *common.HTTP, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["exceptionFilter"]); !ok {
		return
	}

	exceptionFilter := reflect.ValueOf(controller).FieldByName(fieldNameByRole["exceptionFilter"]).Interface().(common.ExceptionFilter)
	items := exceptionFilter.InjectProvidersIntoHTTPExceptionFilters(http, buildFieldInjectionCallback("exceptionFilter", injectedProviders))

	for _, item := range items {
		m.HTTPExceptionFilters = append(m.HTTPExceptionFilters, common.HTTPLayer{
			ControllerPath:  controllerPath,
			Method:          item.HTTP.Method,
			Route:           item.HTTP.Route,
			Version:         item.HTTP.Version,
			Handler:         item.HTTP.Common.Handler,
			Name:            item.HTTP.Common.Name,
			MainHandlerName: item.HTTP.Common.MainHandlerName,
			Pattern:         item.HTTP.Pattern,
		})
	}
}

func (m *Module) bindHTTPMiddlewares(controller Controller, controllerType reflect.Type, controllerPath string, http *common.HTTP, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["middleware"]); !ok {
		return
	}

	middleware := reflect.ValueOf(controller).FieldByName(fieldNameByRole["middleware"]).Interface().(common.Middleware)
	items := middleware.InjectProvidersIntoHTTPMiddlewares(http, buildFieldInjectionCallback("middleware function", injectedProviders))

	for _, item := range items {
		m.HTTPMiddlewares = append(m.HTTPMiddlewares, common.HTTPLayer{
			ControllerPath:  controllerPath,
			Method:          item.HTTP.Method,
			Route:           item.HTTP.Route,
			Version:         item.HTTP.Version,
			Handler:         item.HTTP.Common.Handler,
			Name:            item.HTTP.Common.Name,
			MainHandlerName: item.HTTP.Common.MainHandlerName,
			Pattern:         item.HTTP.Pattern,
		})
	}
}

func (m *Module) bindHTTPGuards(controller Controller, controllerType reflect.Type, controllerPath string, http *common.HTTP, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["guard"]); !ok {
		return
	}

	guard := reflect.ValueOf(controller).FieldByName(fieldNameByRole["guard"]).Interface().(common.Guard)
	items := guard.InjectProvidersIntoHTTPGuards(http, buildFieldInjectionCallback("guarder", injectedProviders))

	for _, item := range items {
		m.HTTPGuards = append(m.HTTPGuards, common.HTTPLayer{
			ControllerPath:  controllerPath,
			Method:          item.HTTP.Method,
			Route:           item.HTTP.Route,
			Version:         item.HTTP.Version,
			Handler:         item.HTTP.Common.Handler,
			Name:            item.HTTP.Common.Name,
			MainHandlerName: item.HTTP.Common.MainHandlerName,
			Pattern:         item.HTTP.Pattern,
		})
	}
}

func (m *Module) bindHTTPInterceptors(controller Controller, controllerType reflect.Type, controllerPath string, http *common.HTTP, injectedProviders map[string]Provider) {
	if _, ok := controllerType.FieldByName(fieldNameByRole["interceptor"]); !ok {
		return
	}

	interceptor := reflect.ValueOf(controller).FieldByName(fieldNameByRole["interceptor"]).Interface().(common.Interceptor)
	items := interceptor.InjectProvidersIntoHTTPInterceptors(http, buildFieldInjectionCallback("interceptor", injectedProviders))

	for _, item := range items {
		m.HTTPInterceptors = append(m.HTTPInterceptors, common.HTTPLayer{
			ControllerPath:  controllerPath,
			Method:          item.HTTP.Method,
			Route:           item.HTTP.Route,
			Version:         item.HTTP.Version,
			Handler:         item.HTTP.Common.Handler,
			Name:            item.HTTP.Common.Name,
			MainHandlerName: item.HTTP.Common.MainHandlerName,
			Pattern:         item.HTTP.Pattern,
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
