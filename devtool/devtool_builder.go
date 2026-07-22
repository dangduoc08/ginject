package devtool

import (
	reflect "reflect"
	"sort"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/internal/slice"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/versioning"
)

type devtoolBuilder struct {
	versioning *versioning.Versioning

	globalExceptionFilters []common.ExceptionFilterable
	globalMiddlewares      []common.MiddlewareFn
	globalGuarders         []common.Guarder
	globalInterceptors     []common.Interceptable

	moduleExceptionFilters []common.HTTPLayer
	moduleMiddlewares      []common.HTTPLayer
	moduleGuarders         []common.HTTPLayer
	moduleInterceptors     []common.HTTPLayer

	exceptionFiltersByPattern map[string][]*common.HTTPLayer
	middlewaresByPattern      map[string][]*common.HTTPLayer
	guardsByPattern           map[string][]*common.HTTPLayer
	interceptorsByPattern     map[string][]*common.HTTPLayer

	httpMainHandlers []common.HTTPLayer
}

func DevtoolBuilder() *devtoolBuilder {
	return &devtoolBuilder{}
}

func (devtoolBuilder *devtoolBuilder) AddExceptionFilters(
	globalExceptionFilters []common.ExceptionFilterable,
	moduleExceptionFilters []common.HTTPLayer,
) *devtoolBuilder {
	devtoolBuilder.globalExceptionFilters = append(devtoolBuilder.globalExceptionFilters, globalExceptionFilters...)
	devtoolBuilder.moduleExceptionFilters = append(devtoolBuilder.moduleExceptionFilters, moduleExceptionFilters...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddMiddlewares(
	globalMiddlewares []common.MiddlewareFn,
	moduleMiddlewares []common.HTTPLayer,
) *devtoolBuilder {
	devtoolBuilder.globalMiddlewares = append(devtoolBuilder.globalMiddlewares, globalMiddlewares...)
	devtoolBuilder.moduleMiddlewares = append(devtoolBuilder.moduleMiddlewares, moduleMiddlewares...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddGuarders(
	globalGuarders []common.Guarder,
	moduleGuarders []common.HTTPLayer,
) *devtoolBuilder {
	devtoolBuilder.globalGuarders = append(devtoolBuilder.globalGuarders, globalGuarders...)
	devtoolBuilder.moduleGuarders = append(devtoolBuilder.moduleGuarders, moduleGuarders...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddInterceptors(
	globalInterceptors []common.Interceptable,
	moduleInterceptors []common.HTTPLayer,
) *devtoolBuilder {
	devtoolBuilder.globalInterceptors = append(devtoolBuilder.globalInterceptors, globalInterceptors...)
	devtoolBuilder.moduleInterceptors = append(devtoolBuilder.moduleInterceptors, moduleInterceptors...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddVersioning(versioning *versioning.Versioning) *devtoolBuilder {
	devtoolBuilder.versioning = versioning

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddHTTPMainHandlers(httpMainHandlers []common.HTTPLayer) *devtoolBuilder {
	devtoolBuilder.httpMainHandlers = append(devtoolBuilder.httpMainHandlers, httpMainHandlers...)

	sort.Slice(devtoolBuilder.httpMainHandlers, func(i, j int) bool {
		return devtoolBuilder.httpMainHandlers[i].Route < devtoolBuilder.httpMainHandlers[j].Route
	})

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) createGlobalHTTPLayers() ([]*Layer, []*Layer, []*Layer, []*Layer) {
	globalExceptionFilters := slice.Map(
		devtoolBuilder.globalExceptionFilters,
		func(el common.ExceptionFilterable, i int) *Layer {
			return &Layer{
				Name:  reflect.TypeOf(el).String(),
				Scope: LayerScope_GLOBAL_SCOPE,
			}
		},
	)

	globalMiddlewares := slice.Map(
		devtoolBuilder.globalMiddlewares,
		func(el common.MiddlewareFn, i int) *Layer {
			return &Layer{
				Name:  reflect.TypeOf(el).String(),
				Scope: LayerScope_GLOBAL_SCOPE,
			}
		},
	)

	globalGuarders := slice.Map(
		devtoolBuilder.globalGuarders,
		func(el common.Guarder, i int) *Layer {
			return &Layer{
				Name:  reflect.TypeOf(el).String(),
				Scope: LayerScope_GLOBAL_SCOPE,
			}
		},
	)

	globalInterceptors := slice.Map(
		devtoolBuilder.globalInterceptors,
		func(el common.Interceptable, i int) *Layer {
			return &Layer{
				Name:  reflect.TypeOf(el).String(),
				Scope: LayerScope_GLOBAL_SCOPE,
			}
		},
	)

	return globalExceptionFilters, globalMiddlewares, globalGuarders, globalInterceptors
}

func (devtoolBuilder *devtoolBuilder) createModuleHTTPLayers(moduleHandlerPattern string) ([]*Layer, []*Layer, []*Layer, []*Layer) {
	moduleExceptionFilters := slice.Map(
		devtoolBuilder.exceptionFiltersByPattern[moduleHandlerPattern],
		func(el *common.HTTPLayer, i int) *Layer {
			return &Layer{
				Name:  el.Name,
				Scope: LayerScope_REQUEST_SCOPE,
			}
		},
	)

	moduleMiddlewares := slice.Map(
		devtoolBuilder.middlewaresByPattern[moduleHandlerPattern],
		func(el *common.HTTPLayer, i int) *Layer {
			return &Layer{
				Name:  el.Name,
				Scope: LayerScope_REQUEST_SCOPE,
			}
		},
	)

	moduleGuards := slice.Map(
		devtoolBuilder.guardsByPattern[moduleHandlerPattern],
		func(el *common.HTTPLayer, i int) *Layer {
			return &Layer{
				Name:  el.Name,
				Scope: LayerScope_REQUEST_SCOPE,
			}
		},
	)

	moduleInterceptors := slice.Map(
		devtoolBuilder.interceptorsByPattern[moduleHandlerPattern],
		func(el *common.HTTPLayer, i int) *Layer {
			return &Layer{
				Name:  el.Name,
				Scope: LayerScope_REQUEST_SCOPE,
			}
		},
	)

	return moduleExceptionFilters, moduleMiddlewares, moduleGuards, moduleInterceptors
}

func (devtoolBuilder *devtoolBuilder) Build() *Devtool {
	devtool := &Devtool{
		GetConfigurationResponse: GetConfigurationResponse{
			Controller: &Controller{
				Http: []*HTTPComponent{},
			},
		},
	}

	globalExceptionFilters,
		globalMiddlewares,
		globalGuards,
		globalInterceptors := devtoolBuilder.createGlobalHTTPLayers()

	devtoolBuilder.exceptionFiltersByPattern = generateLayersByPattern(devtoolBuilder.moduleExceptionFilters)
	devtoolBuilder.middlewaresByPattern = generateLayersByPattern(devtoolBuilder.moduleMiddlewares)
	devtoolBuilder.guardsByPattern = generateLayersByPattern(devtoolBuilder.moduleGuarders)
	devtoolBuilder.interceptorsByPattern = generateLayersByPattern(devtoolBuilder.moduleInterceptors)

	// Create HTTP Component
	for _, moduleHandler := range devtoolBuilder.httpMainHandlers {
		httpMethod := routing.OperationsMapHTTPMethods[moduleHandler.Method]

		moduleExceptionFilters,
			moduleMiddlewares,
			moduleGuards,
			moduleInterceptors := devtoolBuilder.createModuleHTTPLayers(moduleHandler.Pattern)

		httpComponent := &HTTPComponent{
			Handler:          moduleHandler.Name,
			HttpMethod:       httpMethod,
			Route:            moduleHandler.Route,
			ExceptionFilters: append(globalExceptionFilters, moduleExceptionFilters...),
			Middlewares:      append(globalMiddlewares, moduleMiddlewares...),
			Guards:           append(globalGuards, moduleGuards...),
			Interceptors:     append(globalInterceptors, moduleInterceptors...),
			Versioning: &HTTPVersioning{
				Value: moduleHandler.Version,
				Key:   devtoolBuilder.versioning.Key,
				Type:  int32(devtoolBuilder.versioning.Type),
			},
			Request: &HTTPRequest{},
		}

		funcType := reflect.TypeOf(moduleHandler.Handler)

		for i := 0; i < funcType.NumIn(); i++ {
			pipe := funcType.In(i)
			pipeType, schemas := generateRequestPayload(pipe)
			if pipeType != "" {
				switch pipeType {
				case common.BodyPipeableKey:
					httpComponent.Request.Body = schemas
				case common.FormPipeableKey:
					httpComponent.Request.Form = schemas
				case common.QueryPipeableKey:
					httpComponent.Request.Query = schemas
				case common.HeaderPipeableKey:
					httpComponent.Request.Header = schemas
				case common.ParamPipeableKey:
					httpComponent.Request.Param = schemas
				case common.FilePipeableKey:
					httpComponent.Request.File = schemas
				}
			}
		}

		httpComponent.Id = generateHandlerID(moduleHandler.ControllerPath + httpComponent.Handler)
		devtool.Controller.Http = append(devtool.Controller.Http, httpComponent)
	}

	return devtool
}
