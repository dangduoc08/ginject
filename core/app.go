package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/devtool"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/utils"
	"github.com/dangduoc08/ginject/versioning"
	"golang.org/x/net/websocket"
)

type App struct {
	route                                  *routing.Router
	wsEventMap                             map[string][]ctx.Handler // to store WS layers, key = subscribe event name
	wsMainHandlerMap                       map[string]any           // to store WS main handler
	wsEventToID                            sync.Map                 // to store WS ID, key = emit event
	serveStaticMapToLastWildcardSlashIndex map[string]int           // to check public dir URL if has * at last
	module                                 *Module
	ctxPool                                sync.Pool
	globalMiddlewares                      []common.MiddlewareFn
	globalGuarders                         []common.Guarder
	globalInterceptors                     []common.Interceptable
	globalExceptionFilters                 []common.ExceptionFilterable
	injectedProviders                      map[string]Provider
	catchRESTFnsMap                        map[string][]common.Catch
	catchWSFnsMap                          map[string][]common.Catch
	Logger                                 common.Logger
	versioning                             *versioning.Versioning
	isEnableVersioning                     bool
	isEnableDevtool                        bool
	devtool                                *devtool.Devtool
}

// link to aliases
const (
	CONTEXT       = "/*ctx.Context"
	WS_CONNECTION = "/*websocket.Conn"
	REQUEST       = "/*http.Request"
	RESPONSE      = "net/http/http.ResponseWriter"
	BODY          = "github.com/dangduoc08/ginject/ctx/ctx.Body"
	FORM          = "github.com/dangduoc08/ginject/ctx/ctx.Form"
	QUERY         = "github.com/dangduoc08/ginject/ctx/ctx.Query"
	HEADER        = "github.com/dangduoc08/ginject/ctx/ctx.Header"
	PARAM         = "github.com/dangduoc08/ginject/ctx/ctx.Param"
	FILE          = "github.com/dangduoc08/ginject/ctx/ctx.File"
	WS_PAYLOAD    = "github.com/dangduoc08/ginject/ctx/ctx.WSPayload"
	NEXT          = "/func()"
	REDIRECT      = "/func(string)"
)

var dependencies = map[string]int{
	CONTEXT:                    1,
	WS_CONNECTION:              1,
	REQUEST:                    1,
	RESPONSE:                   1,
	BODY:                       1,
	FORM:                       1,
	QUERY:                      1,
	HEADER:                     1,
	PARAM:                      1,
	FILE:                       1,
	WS_PAYLOAD:                 1,
	NEXT:                       1,
	REDIRECT:                   1,
	common.CONTEXT_PIPEABLE:    1,
	common.BODY_PIPEABLE:       1,
	common.FORM_PIPEABLE:       1,
	common.QUERY_PIPEABLE:      1,
	common.HEADER_PIPEABLE:     1,
	common.PARAM_PIPEABLE:      1,
	common.FILE_PIPEABLE:       1,
	common.WS_PAYLOAD_PIPEABLE: 1,
}

var wsPaths = []string{
	"/ws",
	"/ws/",
}

type WithValueKey string

func New() *App {
	event := ctx.NewEvent()

	app := App{
		route:                                  routing.NewRouter(),
		catchRESTFnsMap:                        make(map[string][]common.Catch),
		catchWSFnsMap:                          make(map[string][]common.Catch),
		wsEventMap:                             make(map[string][]func(*ctx.Context)),
		wsMainHandlerMap:                       make(map[string]any),
		serveStaticMapToLastWildcardSlashIndex: make(map[string]int),
		ctxPool: sync.Pool{
			New: func() any {
				c := ctx.NewContext()
				c.Event = event

				return c
			},
		},
	}

	// binding default exception filter
	app.BindGlobalExceptionFilters(globalExceptionFilter{})

	return &app
}

func (app *App) Create(m *Module) {
	if app.Logger == nil {
		app.Logger = log.NewLog(nil)
	}
	globalInterfaces[injectableInterfaces[0]] = app.Logger
	app.module = m.NewModule()

	var injectedProviders map[string]Provider = make(map[string]Provider)
	for _, provider := range app.module.providers {
		injectedProviders[genProviderKey(provider)] = provider
	}
	app.injectedProviders = injectedProviders

	// Request cycles
	// global exception filters
	// module exception filters
	// global middlewares
	// module middlewares
	// global guards
	// module guards
	// global interceptors (pre)
	// module interceptors (pre)
	// main handler

	// REST module exception filters
	totalRESTModuleExceptionFilers := len(app.module.RESTExceptionFilters)
	for i := totalRESTModuleExceptionFilers - 1; i >= 0; i-- {
		moduleExceptionFilter := app.module.RESTExceptionFilters[i]
		httpMethod := routing.OperationsMapHTTPMethods[moduleExceptionFilter.Method]

		endpoint := routing.MethodRouteVersionToPattern(httpMethod, moduleExceptionFilter.Route, moduleExceptionFilter.Version)
		app.catchRESTFnsMap[endpoint] = append(app.catchRESTFnsMap[endpoint], moduleExceptionFilter.Handler.(common.Catch))
	}

	// WS module exception filters
	totalWSModuleExceptionFilers := len(app.module.WSExceptionFilters)
	for i := totalWSModuleExceptionFilers - 1; i >= 0; i-- {
		moduleExceptionFilter := app.module.WSExceptionFilters[i]
		app.catchWSFnsMap[moduleExceptionFilter.EventName] = append(app.catchWSFnsMap[moduleExceptionFilter.EventName], moduleExceptionFilter.Handler.(common.Catch))
	}

	// global exception filters
	totalGlobalExceptionFilters := len(app.globalExceptionFilters)
	for i := totalGlobalExceptionFilters - 1; i >= 0; i-- {
		globalExceptionFilter := app.globalExceptionFilters[i]
		newGlobalExceptionFilter, err := injectDependencies(globalExceptionFilter, "exceptionFilter", injectedProviders)
		if err != nil {
			panic(err)
		}

		globalExceptionFilter = common.Construct(newGlobalExceptionFilter.Interface(), "NewExceptionFilter").(common.ExceptionFilterable)

		// REST global exception filters
		for _, mainHandlerItem := range app.module.RESTMainHandlers {
			httpMethod := routing.OperationsMapHTTPMethods[mainHandlerItem.Method]

			endpoint := routing.MethodRouteVersionToPattern(httpMethod, mainHandlerItem.Route, mainHandlerItem.Version)
			app.catchRESTFnsMap[endpoint] = append(app.catchRESTFnsMap[endpoint], globalExceptionFilter.Catch)
		}

		// WS global exception filters
		for eventName := range common.InsertedEvents {
			app.catchWSFnsMap[eventName] = append(
				app.catchWSFnsMap[eventName],
				globalExceptionFilter.Catch,
			)
		}
	}

	for pattern, catchFns := range app.catchRESTFnsMap {
		catchMiddlewareWrapper := func(catchEvent string, catchFns []common.Catch) ctx.Handler {
			return func(c *ctx.Context) {
				c.Event.Once(catchEvent, func(args ...any) {
					catchFnIndex := args[2].(int)

					defer func() {
						if rec := recover(); rec != nil {
							c.Event.Emit(catchEvent, c, rec, catchFnIndex+1)
						}
					}()

					newC := args[0].(*ctx.Context)
					catchFn := catchFns[catchFnIndex]

					response := http.StatusText(http.StatusInternalServerError)

					switch arg := args[1].(type) {
					case exception.Exception:
						catchFn(newC, &arg)
						return
					case error:
						response = arg.Error()
					case string:
						response = arg
					case int:
					case int8:
					case int16:
					case int32:
					case int64:
					case uint:
					case uint8:
					case uint16:
					case uint32:
					case uint64:
					case float32:
					case float64:
					case complex64:
					case complex128:
					case uintptr:
						response = strconv.Itoa(args[1].(int))
					}
					exception := exception.InternalServerErrorException(response, map[string]any{
						"description": "Unknown exception",
					})
					catchFn(newC, &exception)
				})

				c.Next()
			}
		}(pattern, catchFns)

		// add catch middleware
		method, route, version := routing.PatternToMethodRouteVersion(pattern)
		httpMethod := routing.OperationsMapHTTPMethods[method]

		app.route.For([]string{httpMethod}, route, version)(catchMiddlewareWrapper)
	}

	for pattern, catchFns := range app.catchWSFnsMap {
		catchMiddlewareWrapper := func(catchEvent string, catchFns []common.Catch) ctx.Handler {
			return func(c *ctx.Context) {
				c.Event.Once(catchEvent, func(args ...any) {
					catchFnIndex := args[2].(int)

					defer func() {
						if rec := recover(); rec != nil {
							c.Event.Emit(catchEvent, c, rec, catchFnIndex+1)
						}
					}()

					newC := args[0].(*ctx.Context)
					catchFn := catchFns[catchFnIndex]

					response := http.StatusText(http.StatusInternalServerError)

					switch arg := args[1].(type) {
					case exception.Exception:
						catchFn(newC, &arg)
						return
					case error:
						response = arg.Error()
					case string:
						response = arg
					case int:
					case int8:
					case int16:
					case int32:
					case int64:
					case uint:
					case uint8:
					case uint16:
					case uint32:
					case uint64:
					case float32:
					case float64:
					case complex64:
					case complex128:
					case uintptr:
						response = strconv.Itoa(args[1].(int))
					}
					exception := exception.InternalServerErrorException(response, map[string]any{
						"description": "Unknown exception",
					})
					catchFn(newC, &exception)
				})

				c.Next()
			}
		}(pattern, catchFns)

		// add catch middleware
		app.wsEventMap[pattern] = append(
			app.wsEventMap[pattern],
			catchMiddlewareWrapper,
		)
	}

	// global middlewares
	for _, globalMiddleware := range app.globalMiddlewares {
		newGlobalMiddleware, err := injectDependencies(globalMiddleware, "middleware", injectedProviders)
		if err != nil {
			panic(err)
		}

		globalMiddleware = common.Construct(newGlobalMiddleware.Interface(), "NewMiddleware").(common.MiddlewareFn)

		useMiddlewareWrapper := func(middleware common.MiddlewareFn) ctx.Handler {
			return func(c *ctx.Context) {
				middleware.Use(c, c.Next)
			}
		}(globalMiddleware)

		// REST global guards
		for _, mainHandlerItem := range app.module.RESTMainHandlers {
			httpMethod := routing.OperationsMapHTTPMethods[mainHandlerItem.Method]

			app.route.For([]string{httpMethod}, mainHandlerItem.Route, mainHandlerItem.Version)(useMiddlewareWrapper)
		}

		// WS global guards
		for eventName := range common.InsertedEvents {
			app.wsEventMap[eventName] = append(
				app.wsEventMap[eventName],
				useMiddlewareWrapper,
			)
		}
	}

	// REST module middlewares
	for _, restModuleMiddleware := range app.module.RESTMiddlewares {
		useMiddlewareWrapper := func(useFn common.Use) ctx.Handler {
			return func(c *ctx.Context) {
				useFn(c, c.Next)
			}
		}(restModuleMiddleware.Handler.(common.Use))

		httpMethod := routing.OperationsMapHTTPMethods[restModuleMiddleware.Method]

		app.route.For([]string{httpMethod}, restModuleMiddleware.Route, restModuleMiddleware.Version)(useMiddlewareWrapper)
	}

	// WS module middlewares
	for _, wsModuleMiddleware := range app.module.WSMiddlewares {
		useMiddlewareWrapper := func(useFn common.Use) ctx.Handler {
			return func(c *ctx.Context) {
				useFn(c, c.Next)
			}
		}(wsModuleMiddleware.Handler.(common.Use))

		app.wsEventMap[wsModuleMiddleware.EventName] = append(
			app.wsEventMap[wsModuleMiddleware.EventName],
			useMiddlewareWrapper,
		)
	}

	// global guards
	for _, globalGuard := range app.globalGuarders {
		newGlobalGuard, err := injectDependencies(globalGuard, "guard", injectedProviders)
		if err != nil {
			panic(err)
		}

		globalGuard = common.Construct(newGlobalGuard.Interface(), "NewGuard").(common.Guarder)

		canActivateMiddleware := func(guard common.Guarder) ctx.Handler {
			return func(c *ctx.Context) {
				common.HandleGuard(c, guard.CanActivate(c))
			}
		}(globalGuard)

		// REST global guards
		for _, mainHandlerItem := range app.module.RESTMainHandlers {
			httpMethod := routing.OperationsMapHTTPMethods[mainHandlerItem.Method]

			app.route.For([]string{httpMethod}, mainHandlerItem.Route, mainHandlerItem.Version)(canActivateMiddleware)
		}

		// WS global guards
		for eventName := range common.InsertedEvents {
			app.wsEventMap[eventName] = append(
				app.wsEventMap[eventName],
				canActivateMiddleware,
			)
		}
	}

	// REST module guards
	for _, moduleGuard := range app.module.RESTGuards {
		canActivateMiddlewareWrapper := func(canActiveFn common.CanActivate) ctx.Handler {
			return func(c *ctx.Context) {
				common.HandleGuard(c, canActiveFn(c))
			}
		}(moduleGuard.Handler.(common.CanActivate))

		httpMethod := routing.OperationsMapHTTPMethods[moduleGuard.Method]
		app.route.For([]string{httpMethod}, moduleGuard.Route, moduleGuard.Version)(canActivateMiddlewareWrapper)
	}

	// WS module guards
	for _, moduleGuard := range app.module.WSGuards {

		canActivateMiddlewareWrapper := func(canActiveFn common.CanActivate) ctx.Handler {
			return func(c *ctx.Context) {
				common.HandleGuard(c, canActiveFn(c))
			}
		}(moduleGuard.Handler.(common.CanActivate))

		app.wsEventMap[moduleGuard.EventName] = append(
			app.wsEventMap[moduleGuard.EventName],
			canActivateMiddlewareWrapper,
		)
	}

	// global interceptors
	for _, globalInterceptor := range app.globalInterceptors {
		newGlobalInterceptor, err := injectDependencies(globalInterceptor, "interceptor", injectedProviders)
		if err != nil {
			panic(err)
		}

		globalInterceptor = common.Construct(newGlobalInterceptor.Interface(), "NewInterceptor").(common.Interceptable)

		// REST global interceptors
		for _, mainHandlerItem := range app.module.RESTMainHandlers {
			httpMethod := routing.OperationsMapHTTPMethods[mainHandlerItem.Method]
			endpoint := routing.MethodRouteVersionToPattern(httpMethod, mainHandlerItem.Route, mainHandlerItem.Version)

			interceptMiddleware := func(interceptor common.Interceptable) ctx.Handler {
				return func(c *ctx.Context) {
					aggregationInstance := aggregation.NewAggregation()

					if aggregations, ok := c.Request.Context().Value(WithValueKey(endpoint)).([]*aggregation.Aggregation); ok {
						aggregations = append(aggregations, aggregationInstance)

						newCtx := context.WithValue(c.Request.Context(), WithValueKey(endpoint), aggregations)
						c.Request = c.Request.WithContext(newCtx)
					} else {
						newCtx := context.WithValue(c.Request.Context(), WithValueKey(endpoint), []*aggregation.Aggregation{aggregationInstance})
						c.Request = c.Request.WithContext(newCtx)
					}

					// IsMainHandlerCalled will be = true
					// if Pipe was invoked in Intercept function
					aggregationInstance.IsMainHandlerCalled = false
					aggregationInstance.SetMainData(nil)

					// invoke intercept function
					// value may returned from Pipe function
					// depend on Intercept invoked at run time
					value := interceptor.Intercept(c, aggregationInstance)
					aggregationInstance.InterceptorData = value
					app.setErrorAggregationOperators(c, aggregationInstance)

					c.Next()
				}
			}(globalInterceptor)

			app.route.For([]string{httpMethod}, mainHandlerItem.Route, mainHandlerItem.Version)(interceptMiddleware)
		}

		// WS global interceptors
		for eventName := range common.InsertedEvents {
			interceptMiddleware := func(interceptor common.Interceptable) ctx.Handler {
				return func(c *ctx.Context) {
					aggregationInstance := aggregation.NewAggregation()

					if aggregations, ok := c.Request.Context().Value(WithValueKey(eventName)).([]*aggregation.Aggregation); ok {
						aggregations = append(aggregations, aggregationInstance)

						newCtx := context.WithValue(c.Request.Context(), WithValueKey(eventName), aggregations)
						c.Request = c.Request.WithContext(newCtx)
					} else {
						newCtx := context.WithValue(c.Request.Context(), WithValueKey(eventName), []*aggregation.Aggregation{aggregationInstance})
						c.Request = c.Request.WithContext(newCtx)
					}

					// IsMainHandlerCalled will be = true
					// if Pipe was invoked in Intercept function
					aggregationInstance.IsMainHandlerCalled = false
					aggregationInstance.SetMainData(nil)

					// invoke intercept function
					// value may returned from Pipe function
					// depend on Intercept invoked at run time
					value := interceptor.Intercept(c, aggregationInstance)
					aggregationInstance.InterceptorData = value
					app.setErrorAggregationOperators(c, aggregationInstance)

					c.Next()
				}
			}(globalInterceptor)

			app.wsEventMap[eventName] = append(
				app.wsEventMap[eventName],
				interceptMiddleware,
			)
		}
	}

	// REST module interceptors
	for _, moduleInterceptor := range app.module.RESTInterceptors {
		httpMethod := routing.OperationsMapHTTPMethods[moduleInterceptor.Method]
		endpoint := routing.MethodRouteVersionToPattern(httpMethod, moduleInterceptor.Route, moduleInterceptor.Version)

		interceptMiddleware := func(interceptFn common.Intercept) ctx.Handler {
			return func(c *ctx.Context) {
				aggregationInstance := aggregation.NewAggregation()

				if aggregations, ok := c.Request.Context().Value(WithValueKey(endpoint)).([]*aggregation.Aggregation); ok {
					aggregations = append(aggregations, aggregationInstance)

					newCtx := context.WithValue(c.Request.Context(), WithValueKey(endpoint), aggregations)
					c.Request = c.Request.WithContext(newCtx)
				} else {
					newCtx := context.WithValue(c.Request.Context(), WithValueKey(endpoint), []*aggregation.Aggregation{aggregationInstance})
					c.Request = c.Request.WithContext(newCtx)
				}

				// IsMainHandlerCalled will be = true
				// if Pipe was invoked in Intercept function
				aggregationInstance.IsMainHandlerCalled = false
				aggregationInstance.SetMainData(nil)

				// invoke intercept function
				// value may returned from Pipe function
				// depend on Intercept invoked at run time
				value := interceptFn(c, aggregationInstance)
				aggregationInstance.InterceptorData = value
				app.setErrorAggregationOperators(c, aggregationInstance)

				c.Next()
			}
		}(moduleInterceptor.Handler.(common.Intercept))

		// add interceptor middleware
		app.route.For([]string{httpMethod}, moduleInterceptor.Route, moduleInterceptor.Version)(interceptMiddleware)
	}

	// WS module interceptors
	for _, moduleInterceptor := range app.module.WSInterceptors {
		interceptMiddleware := func(interceptFn common.Intercept) ctx.Handler {
			return func(c *ctx.Context) {
				aggregationInstance := aggregation.NewAggregation()

				if aggregations, ok := c.Request.Context().Value(WithValueKey(moduleInterceptor.EventName)).([]*aggregation.Aggregation); ok {
					aggregations = append(aggregations, aggregationInstance)

					newCtx := context.WithValue(c.Request.Context(), WithValueKey(moduleInterceptor.EventName), aggregations)
					c.Request = c.Request.WithContext(newCtx)
				} else {
					newCtx := context.WithValue(c.Request.Context(), WithValueKey(moduleInterceptor.EventName), []*aggregation.Aggregation{aggregationInstance})
					c.Request = c.Request.WithContext(newCtx)
				}

				// IsMainHandlerCalled will be = true
				// if Pipe was invoked in Intercept function
				aggregationInstance.IsMainHandlerCalled = false
				aggregationInstance.SetMainData(nil)

				// invoke intercept function
				// value may returned from Pipe function
				// depend on Intercept invoked at run time
				value := interceptFn(c, aggregationInstance)
				aggregationInstance.InterceptorData = value
				app.setErrorAggregationOperators(c, aggregationInstance)

				c.Next()
			}
		}(moduleInterceptor.Handler.(common.Intercept))

		app.wsEventMap[moduleInterceptor.EventName] = append(
			app.wsEventMap[moduleInterceptor.EventName],
			interceptMiddleware,
		)
	}

	// main REST handler
	for _, moduleHandler := range app.module.RESTMainHandlers {
		httpMethod := routing.OperationsMapHTTPMethods[moduleHandler.Method]
		if moduleHandler.Method == routing.SERVE {
			r := moduleHandler.Route
			lr := len(r)
			lastWildcardSlashIndex := 0 // zero mean use config dir
			if lr >= 2 && r[lr-2:] == "*/" {
				lastWildcardSlashIndex = strings.Count(r, "/") - 1
			}

			app.serveStaticMapToLastWildcardSlashIndex[routing.MethodRouteVersionToPattern(httpMethod, moduleHandler.Route, moduleHandler.Version)] = lastWildcardSlashIndex
		}
		app.route.AddInjectableHandler(httpMethod, moduleHandler.Route, moduleHandler.Version, moduleHandler.Handler)
	}

	// main WS handler
	for _, moduleHandler := range app.module.WSMainHandlers {
		app.wsMainHandlerMap[moduleHandler.EventName] = moduleHandler.Handler
	}

	if app.isEnableDevtool {
		app.createDevtool()
	}
}

func (app *App) BindGlobalGuards(guarders ...common.Guarder) *App {
	app.globalGuarders = append(app.globalGuarders, guarders...)

	return app
}

func (app *App) BindGlobalInterceptors(interceptors ...common.Interceptable) *App {
	app.globalInterceptors = append(app.globalInterceptors, interceptors...)

	return app
}

func (app *App) BindGlobalExceptionFilters(exceptionFilters ...common.ExceptionFilterable) *App {
	app.globalExceptionFilters = append(app.globalExceptionFilters, exceptionFilters...)

	return app
}

func (app *App) BindGlobalMiddlewares(middlewares ...common.MiddlewareFn) *App {
	app.globalMiddlewares = append(app.globalMiddlewares, middlewares...)

	return app
}

func (app *App) EnableVersioning(v versioning.Versioning) *App {
	app.versioning = &v
	app.isEnableVersioning = true

	return app
}

func (app *App) EnableDevtool() *App {
	app.isEnableDevtool = true

	return app
}

func (app *App) UseLogger(logger common.Logger) *App {
	app.Logger = logger
	globalInterfaces[injectableInterfaces[0]] = app.Logger

	return app
}

func (app *App) Get(p Provider) any {
	k := genProviderKey(p)
	return utils.ArrFind(app.module.providers, func(provider Provider, i int) bool {
		return genProviderKey(provider) == k
	})
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := app.ctxPool.Get().(*ctx.Context)
	c.Timestamp = time.Now()
	c.ResponseWriter = w
	c.Request = r
	ctxID := app.getContextID(c)
	c.SetID(ctxID)

	defer app.ctxPool.Put(c)

	if utils.ArrIncludes(wsPaths, r.URL.Path) {
		c.SetType(ctx.WSType)
		websocket.Handler.ServeHTTP(func(wsConn *websocket.Conn) {
			app.handleWSRequest(wsConn, w, r, c)
		}, w, r)
	} else {
		c.SetType(ctx.HTTPType)
		c.ResponseWriter.Header().Set(ctx.REQUEST_ID, c.GetID())

		app.handleRESTRequest(c)
	}

	c.Reset()
}

func (app *App) Listen(port int) error {

	// REST logs
	routeArr := []string{}
	for r, item := range app.route.Hash {
		if item.HandlerIndex > -1 {
			routeArr = append(routeArr, r)
		}
	}
	sort.Strings(routeArr)

	for _, routeName := range routeArr {
		m, r, v := routing.PatternToMethodRouteVersion(routeName)
		if r == "" {
			r = "/"
		}
		args := []any{"method", m, "route", r}
		if v != "" {
			args = append(args, "version", v)
		}
		app.Logger.Info(
			"RouteExplorer",
			args...,
		)
	}

	// WS logs
	eventArr := []string{}
	for e := range common.InsertedEvents {
		eventArr = append(eventArr, e)
	}
	sort.Strings(eventArr)

	for _, eventName := range eventArr {
		p, e := ctx.ResolveWSEventname(eventName)

		app.Logger.Info(
			"WebSocketEvent",
			"subprotocol", p,
			"subscribe", e,
		)
	}

	addr := fmt.Sprintf(":%v", port)
	logBoostrap(port)
	return http.ListenAndServe(addr, app)
}

func (app *App) handleRESTRequest(c *ctx.Context) {
	var catchEvent string

	defer func() {
		if rec := recover(); rec != nil {

			// Pipe errors run first
			// then exception filter
			if errorAggregationOperators, ok := c.Request.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY)).([]aggregation.AggregationOperator); ok {
				totalErrorAggregations := len(errorAggregationOperators)

				// Handle case if pipe error panic
				defer func() {
					if rec := recover(); rec != nil {
						c.Event.Emit(catchEvent, c, rec, 0)
					}
				}()

				for i := totalErrorAggregations - 1; i >= 0; i-- {
					aggregation := errorAggregationOperators[i]
					rec = aggregation(c, rec)
				}
			}

			// Execute exception filters if any
			// normally this one always ok
			// since we always set global exception filter as default
			if _, ok := app.catchRESTFnsMap[catchEvent]; ok && rec != nil {

				// 3rd param is index of catch function
				c.Event.Emit(catchEvent, c, rec, 0)
			}
		}
	}()

	isNext := true
	c.Next = func() {
		isNext = true
	}

	version := ""
	if app.isEnableVersioning {
		version = app.versioning.GetVersion(c)
	}

	isMatched, matchedRoute, paramKeys, paramValues, handlers := app.route.Match(c.Request.Method, c.Request.URL.Path, version)
	if !isMatched {
		isMatched, matchedRoute, paramKeys, paramValues, handlers = app.route.Match(c.Request.Method, c.Request.URL.Path, versioning.NEUTRAL_VERSION)
	}

	if app.isEnableVersioning {
		if version == "" && isMatched {
			// Invoke middlewares
			for _, middleware := range app.route.GlobalMiddlewares {
				if isNext {
					isNext = false
					middleware(c)
				}
			}

			if isNext {
				app.returnDeprecatedURL(c)
			}

			return
		}
	}

	catchEvent = matchedRoute

	if isMatched {
		c.SetRoute(matchedRoute)
		c.ParamKeys = paramKeys
		c.ParamValues = paramValues
		if c.Request.Method == http.MethodPost {
			c.Status(http.StatusCreated)
		}

		for _, handler := range handlers {
			if isNext {
				isNext = false
				if handler == nil {

					// handler = nil / main handler
					// meaning this is injectable handler
					injectableHandler := app.route.InjectableHandlers[matchedRoute]

					// data return from main handler
					data := app.provideAndInvoke(injectableHandler, c)

					if aggregations, ok := c.Request.Context().Value(WithValueKey(matchedRoute)).([]*aggregation.Aggregation); ok {
						var aggregatedData any
						isMainHandlerCalled := true

						totalAggregations := len(aggregations)

						for i := totalAggregations - 1; i >= 0; i-- {
							aggregation := aggregations[i]

							if aggregation.IsMainHandlerCalled {

								// set data from main handler into
								// first interceptor
								if i == totalAggregations-1 {
									if len(data) == 1 {
										aggregatedData = data[0].Interface()
									} else if len(data) > 1 {
										setStatusCode(c, data[0])
										aggregatedData = data[1].Interface()
									}
								}

								aggregation.SetMainData(aggregatedData)
								aggregatedData = aggregation.Aggregate(c)
							} else {
								isMainHandlerCalled = false
								if lastWildcardSlashIndex, ok := app.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
									var dir any

									if len(data) == 1 {
										dir = data[0].Interface()
									} else if len(data) > 1 {
										setStatusCode(c, data[0])
										dir = data[1].Interface()
									}
									app.serveContent(c, lastWildcardSlashIndex, dir)
								} else {
									returnREST(c, reflect.ValueOf(aggregation.InterceptorData))
								}
								break
							}
						}

						if isMainHandlerCalled {
							if lastWildcardSlashIndex, ok := app.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
								var dir any

								if len(data) == 1 {
									dir = data[0].Interface()
								} else if len(data) > 1 {
									setStatusCode(c, data[0])
									dir = data[1].Interface()
								}
								app.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnREST(c, reflect.ValueOf(aggregatedData))
							}
						}
					} else {
						if len(data) == 1 {
							if lastWildcardSlashIndex, ok := app.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
								dir := data[0].Interface()
								app.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnREST(c, data[0])
							}
						} else if len(data) > 1 {
							setStatusCode(c, data[0])
							if lastWildcardSlashIndex, ok := app.serveStaticMapToLastWildcardSlashIndex[matchedRoute]; ok {
								dir := data[1].Interface()
								app.serveContent(c, lastWildcardSlashIndex, dir)
							} else {
								returnREST(c, data[1])
							}
						}
					}
				} else {
					handler(c)
				}
			}
		}
	} else {
		// Invoke middlewares
		for _, middleware := range app.route.GlobalMiddlewares {
			if isNext {
				isNext = false
				middleware(c)
			}
		}

		if isNext {
			app.returnNotFound(c)
		}
	}
}

func (app *App) handleWSRequest(wsConn *websocket.Conn, w http.ResponseWriter, r *http.Request, c *ctx.Context) {
	wsInstance := ctx.NewWS(wsConn)
	c.WS = wsInstance
	isNext := true
	c.Next = func() {
		isNext = true
	}
	wsid := wsInstance.GetConnID()
	wsSubscribedEvents := wsInstance.GetSubscribedEvents()

	defer func() {
		for _, subscribedEventName := range wsSubscribedEvents {
			app.removeWSEvent(subscribedEventName, wsid, c)
		}
		wsConn.Close()
	}()

	if !wsInstance.CanEstablish(common.InsertedEvents) {
		return
	}

	for _, subscribedEventName := range wsSubscribedEvents {
		app.addWSEvent(subscribedEventName, wsid, c, func(args ...any) {
			wsInstance.SendToConn(c, wsConn, args[0].(string))
		})
	}

	for {

		// listen on comming messages
		var message []byte
		err := websocket.Message.Receive(wsConn, &message)

		// reset timestamp
		// based on time when receive message
		c.Timestamp = time.Now()

		if err != nil {

			// client close connection
			if err == io.EOF {
				break
			}
			app.wsInvokeMiddlewares(c, exception.UnsupportedMediaTypeException(err.Error()))
			continue
		}

		var wsMsg ctx.WSMessage
		err = json.Unmarshal(message, &wsMsg)
		if err != nil {
			app.wsInvokeMiddlewares(c, exception.UnsupportedMediaTypeException(err.Error()))
			continue
		}

		// event was registered by controller
		var publishEventName string
		defer func() {
			if rec := recover(); rec != nil {

				// Pipe errors run first
				// then exception filter
				if errorAggregationOperators, ok := c.Request.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY)).([]aggregation.AggregationOperator); ok {
					totalErrorAggregations := len(errorAggregationOperators)

					// Handle case if pipe error panic
					defer func() {
						if rec := recover(); rec != nil {
							c.Event.Emit(publishEventName, c, rec, 0)
						}
					}()

					for i := totalErrorAggregations - 1; i >= 0; i-- {
						aggregation := errorAggregationOperators[i]
						rec = aggregation(c, rec)
					}
				}

				// Execute exception filters if any
				// normally this one always ok
				// since we always set global exception filter as default
				if _, ok := app.catchWSFnsMap[publishEventName]; ok && rec != nil {

					// 3rd param is index of catch function
					c.Event.Emit(publishEventName, c, rec, 0)
				}

				// reset ErrorAggregationOperators
				// to prevent duplicate error aggregation
				// due to error will be added
				// whenever interceptor triggered
				// but WS 1 connection use 1 ctx
				newCtx := context.WithValue(c.Request.Context(), WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY), nil)
				c.Request = c.Request.WithContext(newCtx)

				// clean all events before recursion
				// prevent emit duplicate event
				for _, eventName := range wsSubscribedEvents {
					app.removeWSEvent(eventName, wsid, c)
				}

				// recursion to keep connection alive
				app.handleWSRequest(wsConn, w, r, c)
			}
		}()

		c.WS.Message = wsMsg
		publishEventName = common.ToWSEventName(wsInstance.GetSubprotocol(), wsMsg.Event)

		if handlers, isMatched := app.wsEventMap[publishEventName]; isMatched {
			for index, handler := range handlers {
				if isNext {
					isNext = false
					handler(c)

					// when ran through all middlewares
					// then invoke mainhandler
					if index == len(handlers)-1 && isNext {
						injectableHandler := app.wsMainHandlerMap[publishEventName]

						// data return from main handler
						data := app.provideAndInvoke(injectableHandler, c)
						if len(data) == 1 {
							data = append(data, reflect.ValueOf("*"))
							data[1], data[0] = data[0], data[1]
						}
						configPublishedEventName := data[0].String()

						if aggregations, ok := c.Request.Context().Value(WithValueKey(publishEventName)).([]*aggregation.Aggregation); ok {
							var aggregatedData any
							isMainHandlerCalled := true

							totalAggregations := len(aggregations)

							for i := totalAggregations - 1; i >= 0; i-- {
								aggregation := aggregations[i]

								if aggregation.IsMainHandlerCalled {

									// set data from main handler into
									// first interceptor
									if i == totalAggregations-1 && len(data) > 1 {
										aggregatedData = data[1].Interface()
									}

									aggregation.SetMainData(aggregatedData)
									aggregatedData = aggregation.Aggregate(c)
								} else {
									isMainHandlerCalled = false
									wsMsg := toWSMessage(reflect.ValueOf(aggregation.InterceptorData))
									app.publishWSEvent(configPublishedEventName, wsMsg, c)
									break
								}
							}

							if isMainHandlerCalled {
								wsMsg := toWSMessage(reflect.ValueOf(aggregatedData))
								app.publishWSEvent(configPublishedEventName, wsMsg, c)
							}
						} else {
							if len(data) > 1 {
								wsMsg := toWSMessage(data[1])
								app.publishWSEvent(configPublishedEventName, wsMsg, c)
							}
						}
					}
				}
			}
		} else {
			app.wsInvokeMiddlewares(c, exception.NotFoundException(fmt.Sprintf("Cannot emit %v event", wsMsg.Event)))
		}
	}
}

func (app *App) provideAndInvoke(f any, c *ctx.Context) []reflect.Value {
	args := []reflect.Value{}
	getFnArgs(f, app.injectedProviders, func(dynamicArgKey string, i int, pipeValue reflect.Value) {
		if _, ok := dependencies[dynamicArgKey]; ok {
			args = append(args, reflect.ValueOf(getDependency(dynamicArgKey, c, pipeValue)))
		} else {
			panic(fmt.Errorf(
				"can't resolve dependencies of the %v. Please make sure that the argument dependency at index [%v] is available in the handler",
				reflect.TypeOf(f).String(),
				i,
			))
		}
	})

	return reflect.ValueOf(f).Call(args)
}

func (app *App) addWSEvent(subscribedEventName, wsid string, c *ctx.Context, cb func(args ...any)) {

	// actual event = eventName + Sec-Websocket-Key + uuid
	c.Event.On(subscribedEventName+wsid, cb)
	if wsids, ok := app.wsEventToID.Load(subscribedEventName); ok {
		wsids := wsids.([]string)
		wsids = append(wsids, wsid)
		app.wsEventToID.Store(subscribedEventName, wsids)
	} else {
		app.wsEventToID.Store(subscribedEventName, []string{wsid})
	}
}

func (app *App) removeWSEvent(subscribedEventName, wsid string, c *ctx.Context) {
	c.Event.RemoveAllListeners(subscribedEventName + wsid)
	if wsids, ok := app.wsEventToID.Load(subscribedEventName); ok {
		wsids = utils.ArrFilter(wsids.([]string), func(el string, i int) bool {
			return el != wsid
		})
		app.wsEventToID.Swap(subscribedEventName, wsids)
	}
}

func (app *App) publishWSEvent(configPublishedEventName, wsMsg string, c *ctx.Context) {
	app.wsEventToID.Range(func(subscribedEventName, wsids any) bool {
		if subscribedEventName == configPublishedEventName {
			for _, wsid := range wsids.([]string) {
				c.Event.Emit(configPublishedEventName+wsid, wsMsg)
			}
		}
		return true
	})

	// reset ErrorAggregationOperators
	// to prevent duplicate error aggregation
	// due to error will be added
	// whenever interceptor triggered
	// but WS 1 connection use 1 ctx
	newCtx := context.WithValue(c.Request.Context(), WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY), nil)
	c.Request = c.Request.WithContext(newCtx)
}

func (app *App) wsInvokeMiddlewares(c *ctx.Context, exception exception.Exception) {
	isNext := true
	c.Next = func() {
		isNext = true
	}

	for _, globalMiddleware := range app.globalMiddlewares {
		if isNext {
			isNext = false
			globalMiddleware.Use(c, c.Next)
		}
	}

	if isNext {
		c.WS.SendSelf(c, ctx.Map{
			"code":    exception.GetCode(),
			"error":   exception.Error(),
			"message": exception.GetResponse(),
		})
	}
}

func (app *App) getContextID(c *ctx.Context) string {
	reqID := c.Header().Get(ctx.REQUEST_ID)
	if reqID == "" {
		uuid, _ := utils.StrUUID()
		return uuid
	}

	return reqID
}

func (app *App) returnNotFound(c *ctx.Context) {
	notFoundException := exception.NotFoundException(fmt.Sprintf("Cannot %v %v", c.Method, c.URL.Path))
	httpCode, _ := notFoundException.GetHTTPStatus()
	c.Status(httpCode)
	c.JSON(ctx.Map{
		"code":    notFoundException.GetCode(),
		"error":   notFoundException.Error(),
		"message": notFoundException.GetResponse(),
	})
}

func (app *App) returnInvalidURL(c *ctx.Context) {
	badRequestException := exception.BadRequestException("Invalid URL path")
	httpCode, _ := badRequestException.GetHTTPStatus()
	c.Status(httpCode)
	c.JSON(ctx.Map{
		"code":    badRequestException.GetCode(),
		"error":   badRequestException.Error(),
		"message": badRequestException.GetResponse(),
	})
}

func (app *App) returnDeprecatedURL(c *ctx.Context) {
	goneException := exception.GoneException("Deprecated URL usage")
	httpCode, _ := goneException.GetHTTPStatus()
	c.Status(httpCode)
	c.JSON(ctx.Map{
		"code":    goneException.GetCode(),
		"error":   goneException.Error(),
		"message": goneException.GetResponse(),
	})
}

func (app *App) setErrorAggregationOperators(c *ctx.Context, aggregationInstance *aggregation.Aggregation) {
	errorAggregationOpr := aggregationInstance.GetAggregationOperator(aggregation.OPERATOR_ERROR)
	if errorAggregationOpr != nil {
		errorAggregationOperators := c.Request.Context().Value(WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY))
		if errorAggregationOperators == nil {
			errorAggregationOperators = []aggregation.AggregationOperator{}
		}
		errorAggregationOperators = append(errorAggregationOperators.([]aggregation.AggregationOperator), errorAggregationOpr)

		newCtx := context.WithValue(c.Request.Context(), WithValueKey(aggregation.ERROR_AGGREGATION_CTX_VALUE_KEY), errorAggregationOperators)
		c.Request = c.Request.WithContext(newCtx)
	}
}

func (app *App) serveContent(c *ctx.Context, lastWildcardSlashIndex int, dir any) {
	if dir, ok := dir.(string); ok {
		if lastWildcardSlashIndex != 0 {
			urlPath := utils.StrRemoveDup(c.Request.URL.Path, "/")
			urlPathArr := strings.Split(urlPath, "/")
			suffix := strings.Join(urlPathArr[lastWildcardSlashIndex:], "/")
			oldDir := dir
			dir = path.Join(dir, suffix)

			if len(dir) < len(oldDir) {
				app.returnInvalidURL(c)
				return
			}
		}

		if _, err := os.Stat(dir); os.IsNotExist(err) || err != nil {
			app.returnNotFound(c)
		} else {
			http.ServeFile(c.ResponseWriter, c.Request, dir)
			c.Event.Emit(ctx.REQUEST_FINISHED, c)
		}
	} else {
		app.returnNotFound(c)
	}
}

func (app *App) createDevtool() {
	devtoolBuilder := devtool.DevtoolBuilder()

	app.devtool = devtoolBuilder.
		AddExceptionFilters(app.globalExceptionFilters, app.module.RESTExceptionFilters).
		AddMiddlewares(app.globalMiddlewares, app.module.RESTMiddlewares).
		AddGuarders(app.globalGuarders, app.module.RESTGuards).
		AddInterceptors(app.globalInterceptors, app.module.RESTInterceptors).
		AddVersioning(app.versioning).
		AddRESTMainHandlers(app.module.RESTMainHandlers).
		Build()

	go app.devtool.Serve()
}
