package devtool

import (
	reflect "reflect"
	"sort"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/utils"
	"github.com/dangduoc08/ginject/versioning"
)

type devtoolBuilder struct {
	versioning *versioning.Versioning

	globalExceptionFilters []common.ExceptionFilterable
	globalMiddlewares      []common.MiddlewareFn
	globalGuarders         []common.Guarder
	globalInterceptors     []common.Interceptable

	moduleExceptionFilters []common.RESTLayer
	moduleMiddlewares      []common.RESTLayer
	moduleGuarders         []common.RESTLayer
	moduleInterceptors     []common.RESTLayer

	exceptionFiltersByPattern map[string][]*common.RESTLayer
	middlewaresByPattern      map[string][]*common.RESTLayer
	guardsByPattern           map[string][]*common.RESTLayer
	interceptorsByPattern     map[string][]*common.RESTLayer

	restMainHandlers []common.RESTLayer
}

func DevtoolBuilder() *devtoolBuilder {
	return &devtoolBuilder{}
}

func (devtoolBuilder *devtoolBuilder) AddExceptionFilters(
	globalExceptionFilters []common.ExceptionFilterable,
	moduleExceptionFilters []common.RESTLayer,
) *devtoolBuilder {
	devtoolBuilder.globalExceptionFilters = append(devtoolBuilder.globalExceptionFilters, globalExceptionFilters...)
	devtoolBuilder.moduleExceptionFilters = append(devtoolBuilder.moduleExceptionFilters, moduleExceptionFilters...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddMiddlewares(
	globalMiddlewares []common.MiddlewareFn,
	moduleMiddlewares []common.RESTLayer,
) *devtoolBuilder {
	devtoolBuilder.globalMiddlewares = append(devtoolBuilder.globalMiddlewares, globalMiddlewares...)
	devtoolBuilder.moduleMiddlewares = append(devtoolBuilder.moduleMiddlewares, moduleMiddlewares...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddGuarders(
	globalGuarders []common.Guarder,
	moduleGuarders []common.RESTLayer,
) *devtoolBuilder {
	devtoolBuilder.globalGuarders = append(devtoolBuilder.globalGuarders, globalGuarders...)
	devtoolBuilder.moduleGuarders = append(devtoolBuilder.moduleGuarders, moduleGuarders...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddInterceptors(
	globalInterceptors []common.Interceptable,
	moduleInterceptors []common.RESTLayer,
) *devtoolBuilder {
	devtoolBuilder.globalInterceptors = append(devtoolBuilder.globalInterceptors, globalInterceptors...)
	devtoolBuilder.moduleInterceptors = append(devtoolBuilder.moduleInterceptors, moduleInterceptors...)

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddVersioning(versioning *versioning.Versioning) *devtoolBuilder {
	devtoolBuilder.versioning = versioning

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) AddRESTMainHandlers(restMainHandlers []common.RESTLayer) *devtoolBuilder {
	devtoolBuilder.restMainHandlers = append(devtoolBuilder.restMainHandlers, restMainHandlers...)

	sort.Slice(devtoolBuilder.restMainHandlers, func(i, j int) bool {
		return devtoolBuilder.restMainHandlers[i].Route < devtoolBuilder.restMainHandlers[j].Route
	})

	return devtoolBuilder
}

func (devtoolBuilder *devtoolBuilder) createGlobalRESTLayers() ([]*Layer, []*Layer, []*Layer, []*Layer) {
	globalExceptionFilters := utils.ArrMap(
		devtoolBuilder.globalExceptionFilters,
		func(el common.ExceptionFilterable, i int) *Layer {
			return &Layer{
				Name:  reflect.TypeOf(el).String(),
				Scope: LayerScope_GLOBAL_SCOPE,
			}
		},
	)

	globalMiddlewares := utils.ArrMap(
		devtoolBuilder.globalMiddlewares,
		func(el common.MiddlewareFn, i int) *Layer {
			return &Layer{
				Name:  reflect.TypeOf(el).String(),
				Scope: LayerScope_GLOBAL_SCOPE,
			}
		},
	)

	globalGuarders := utils.ArrMap(
		devtoolBuilder.globalGuarders,
		func(el common.Guarder, i int) *Layer {
			return &Layer{
				Name:  reflect.TypeOf(el).String(),
				Scope: LayerScope_GLOBAL_SCOPE,
			}
		},
	)

	globalInterceptors := utils.ArrMap(
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

func (devtoolBuilder *devtoolBuilder) createModuleRESTLayers(moduleHandlerPattern string) ([]*Layer, []*Layer, []*Layer, []*Layer) {
	moduleExceptionFilters := utils.ArrMap(
		devtoolBuilder.exceptionFiltersByPattern[moduleHandlerPattern],
		func(el *common.RESTLayer, i int) *Layer {
			return &Layer{
				Name:  el.Name,
				Scope: LayerScope_REQUEST_SCOPE,
			}
		},
	)

	moduleMiddlewares := utils.ArrMap(
		devtoolBuilder.middlewaresByPattern[moduleHandlerPattern],
		func(el *common.RESTLayer, i int) *Layer {
			return &Layer{
				Name:  el.Name,
				Scope: LayerScope_REQUEST_SCOPE,
			}
		},
	)

	moduleGuards := utils.ArrMap(
		devtoolBuilder.guardsByPattern[moduleHandlerPattern],
		func(el *common.RESTLayer, i int) *Layer {
			return &Layer{
				Name:  el.Name,
				Scope: LayerScope_REQUEST_SCOPE,
			}
		},
	)

	moduleInterceptors := utils.ArrMap(
		devtoolBuilder.interceptorsByPattern[moduleHandlerPattern],
		func(el *common.RESTLayer, i int) *Layer {
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
				Rest: []*RESTComponent{},
			},
		},
	}

	globalExceptionFilters,
		globalMiddlewares,
		globalGuards,
		globalInterceptors := devtoolBuilder.createGlobalRESTLayers()

	devtoolBuilder.exceptionFiltersByPattern = generateLayersByPattern(devtoolBuilder.moduleExceptionFilters)
	devtoolBuilder.middlewaresByPattern = generateLayersByPattern(devtoolBuilder.moduleMiddlewares)
	devtoolBuilder.guardsByPattern = generateLayersByPattern(devtoolBuilder.moduleGuarders)
	devtoolBuilder.interceptorsByPattern = generateLayersByPattern(devtoolBuilder.moduleInterceptors)

	// Create REST Component
	for _, moduleHandler := range devtoolBuilder.restMainHandlers {
		httpMethod := routing.OperationsMapHTTPMethods[moduleHandler.Method]

		moduleExceptionFilters,
			moduleMiddlewares,
			moduleGuards,
			moduleInterceptors := devtoolBuilder.createModuleRESTLayers(moduleHandler.Pattern)

		restComponent := &RESTComponent{
			Handler:          moduleHandler.Name,
			HttpMethod:       httpMethod,
			Route:            moduleHandler.Route,
			ExceptionFilters: append(globalExceptionFilters, moduleExceptionFilters...),
			Middlewares:      append(globalMiddlewares, moduleMiddlewares...),
			Guards:           append(globalGuards, moduleGuards...),
			Interceptors:     append(globalInterceptors, moduleInterceptors...),
			Versioning: &RESTVersioning{
				Value: moduleHandler.Version,
				Key:   devtoolBuilder.versioning.Key,
				Type:  int32(devtoolBuilder.versioning.Type),
			},
			Request: &RESTRequest{},
		}

		funcType := reflect.TypeOf(moduleHandler.Handler)

		for i := 0; i < funcType.NumIn(); i++ {
			pipe := funcType.In(i)
			pipeType, schemas := generateRequestPayload(pipe)
			if pipeType != "" {
				switch pipeType {
				case common.BODY_PIPEABLE:
					restComponent.Request.Body = schemas
				case common.FORM_PIPEABLE:
					restComponent.Request.Form = schemas
				case common.QUERY_PIPEABLE:
					restComponent.Request.Query = schemas
				case common.HEADER_PIPEABLE:
					restComponent.Request.Header = schemas
				case common.PARAM_PIPEABLE:
					restComponent.Request.Param = schemas
				case common.FILE_PIPEABLE:
					restComponent.Request.File = schemas
				}
			}
		}

		restComponent.Id = generateHandlerID(moduleHandler.ControllerPath + restComponent.Handler)
		devtool.Controller.Rest = append(devtool.Controller.Rest, restComponent)
	}

	return devtool
}
