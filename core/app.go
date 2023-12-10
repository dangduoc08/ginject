package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"time"

	stdContext "context"

	"github.com/dangduoc08/gooh/aggregation"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/exception"
	"github.com/dangduoc08/gooh/log"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
	"golang.org/x/net/websocket"
)

type globalMiddleware struct {
	route   string
	handler context.Handler
}

type App struct {
	route                  *routing.Router
	wsEventMap             map[string][]context.Handler // to store WS layers, key = subscribe event name
	wsMainHandlerMap       map[string]any               // to store WS main handler
	wsEventToID            sync.Map                     // to store WS ID, key = emit event
	module                 *Module
	ctxPool                sync.Pool
	globalMiddlewares      []globalMiddleware
	globalGuarders         []common.Guarder
	globalInterceptors     []common.Interceptable
	globalExceptionFilters []common.ExceptionFilterable
	injectedProviders      map[string]Provider
	restAggregationMap     map[string][]*aggregation.Aggregation
	wsAggregationMap       map[string][]*aggregation.Aggregation
	catchRESTFnsMap        map[string][]common.Catch
	catchWSFnsMap          map[string][]common.Catch
	Logger                 common.Logger
}

// link to aliases
const (
	CONTEXT             = "/*context.Context"
	WS_CONNECTION       = "/*websocket.Conn"
	REQUEST             = "/*http.Request"
	RESPONSE            = "net/http/http.ResponseWriter"
	BODY                = "github.com/dangduoc08/gooh/context/context.Body"
	FORM                = "github.com/dangduoc08/gooh/context/context.Form"
	QUERY               = "github.com/dangduoc08/gooh/context/context.Query"
	HEADER              = "github.com/dangduoc08/gooh/context/context.Header"
	PARAM               = "github.com/dangduoc08/gooh/context/context.Param"
	WS_PAYLOAD          = "github.com/dangduoc08/gooh/context/context.WSPayload"
	NEXT                = "/func()"
	REDIRECT            = "/func(string)"
	CONTEXT_PIPEABLE    = "context"
	BODY_PIPEABLE       = "body"
	FORM_PIPEABLE       = "form"
	QUERY_PIPEABLE      = "query"
	HEADER_PIPEABLE     = "header"
	PARAM_PIPEABLE      = "param"
	WS_PAYLOAD_PIPEABLE = "wsPayload"
)

var dependencies = map[string]int{
	CONTEXT:             1,
	WS_CONNECTION:       1,
	REQUEST:             1,
	RESPONSE:            1,
	BODY:                1,
	FORM:                1,
	QUERY:               1,
	HEADER:              1,
	PARAM:               1,
	WS_PAYLOAD:          1,
	NEXT:                1,
	REDIRECT:            1,
	CONTEXT_PIPEABLE:    1,
	BODY_PIPEABLE:       1,
	FORM_PIPEABLE:       1,
	QUERY_PIPEABLE:      1,
	HEADER_PIPEABLE:     1,
	PARAM_PIPEABLE:      1,
	WS_PAYLOAD_PIPEABLE: 1,
}

var wsPaths = []string{
	"/ws",
	"/ws/",
}

type WithValueKey string

func New() *App {
	event := context.NewEvent()

	app := App{
		route:              routing.NewRouter(),
		restAggregationMap: make(map[string][]*aggregation.Aggregation),
		wsAggregationMap:   make(map[string][]*aggregation.Aggregation),
		catchRESTFnsMap:    make(map[string][]common.Catch),
		catchWSFnsMap:      make(map[string][]common.Catch),
		wsEventMap:         make(map[string][]func(*context.Context)),
		wsMainHandlerMap:   make(map[string]any),
		ctxPool: sync.Pool{
			New: func() any {
				c := context.NewContext()
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
		endpoint := routing.ToEndpoint(routing.AddMethodToRoute(moduleExceptionFilter.Route, moduleExceptionFilter.Method))
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
			endpoint := routing.ToEndpoint(routing.AddMethodToRoute(mainHandlerItem.Route, mainHandlerItem.Method))
			app.catchRESTFnsMap[endpoint] = append(app.catchRESTFnsMap[endpoint], globalExceptionFilter.Catch)
		}

		// WS global exception filters
		for eventName := range insertedEvents {
			app.catchWSFnsMap[eventName] = append(
				app.catchWSFnsMap[eventName],
				globalExceptionFilter.Catch,
			)
		}
	}

	for pattern, catchFns := range app.catchRESTFnsMap {
		catchMiddleware := func(catchEvent string, catchFns []common.Catch) context.Handler {
			return func(ctx *context.Context) {
				ctx.Event.Once(catchEvent, func(args ...any) {
					catchFnIndex := args[2].(int)

					defer func() {
						if rec := recover(); rec != nil {
							ctx.Event.Emit(catchEvent, ctx, rec, catchFnIndex+1)
						}
					}()

					newC := args[0].(*context.Context)
					catchFn := catchFns[catchFnIndex]

					response := http.StatusText(http.StatusInternalServerError)

					switch arg := args[1].(type) {
					case exception.HTTPException:
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
					httpException := exception.InternalServerErrorException(response, map[string]any{
						"description": "Unknown exception",
					})
					catchFn(newC, &httpException)
				})

				ctx.Next()
			}
		}(pattern, catchFns)

		// add catch middleware
		httpMethod, route := routing.SplitRoute(pattern)
		app.route.For(route, []string{httpMethod})(catchMiddleware)
	}

	for pattern, catchFns := range app.catchWSFnsMap {
		catchMiddleware := func(catchEvent string, catchFns []common.Catch) context.Handler {
			return func(ctx *context.Context) {
				ctx.Event.Once(catchEvent, func(args ...any) {
					catchFnIndex := args[2].(int)

					defer func() {
						if rec := recover(); rec != nil {
							ctx.Event.Emit(catchEvent, ctx, rec, catchFnIndex+1)
						}
					}()

					newC := args[0].(*context.Context)
					catchFn := catchFns[catchFnIndex]

					response := http.StatusText(http.StatusInternalServerError)

					switch arg := args[1].(type) {
					case exception.HTTPException:
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
					httpException := exception.InternalServerErrorException(response, map[string]any{
						"description": "Unknown exception",
					})
					catchFn(newC, &httpException)
				})

				ctx.Next()
			}
		}(pattern, catchFns)

		// add catch middleware
		app.wsEventMap[pattern] = append(
			app.wsEventMap[pattern],
			catchMiddleware,
		)
	}

	// global middlewares
	for _, globalMiddleware := range app.globalMiddlewares {
		if globalMiddleware.route != "*" {
			app.route.For(globalMiddleware.route, routing.HTTPMethods)(globalMiddleware.handler)
		} else {

			// REST global middlewares
			app.route.Use(globalMiddleware.handler)

			// WS global middlewares
			for eventName := range insertedEvents {
				app.wsEventMap[eventName] = append(
					app.wsEventMap[eventName],
					globalMiddleware.handler,
				)
			}
		}
	}

	// REST module middlewares
	for _, restModuleMiddleware := range app.module.RESTMiddlewares {
		app.route.For(restModuleMiddleware.Route, []string{restModuleMiddleware.Method})(restModuleMiddleware.Handlers...)
	}

	// WS module middlewares
	for _, wsModuleMiddleware := range app.module.WSMiddlewares {
		app.wsEventMap[wsModuleMiddleware.EventName] = append(
			app.wsEventMap[wsModuleMiddleware.EventName],
			wsModuleMiddleware.Handlers...,
		)
	}

	// global guards
	for _, globalGuard := range app.globalGuarders {
		newGlobalGuard, err := injectDependencies(globalGuard, "guard", injectedProviders)
		if err != nil {
			panic(err)
		}

		globalGuard = common.Construct(newGlobalGuard.Interface(), "NewGuard").(common.Guarder)

		canActivateMiddleware := func(guard common.Guarder) context.Handler {
			return func(ctx *context.Context) {
				common.HandleGuard(ctx, guard.CanActivate(ctx))
			}
		}(globalGuard)

		// REST global guards
		for _, mainHandlerItem := range app.module.RESTMainHandlers {
			app.route.For(mainHandlerItem.Route, []string{mainHandlerItem.Method})(canActivateMiddleware)
		}

		// WS global guards
		for eventName := range insertedEvents {
			app.wsEventMap[eventName] = append(
				app.wsEventMap[eventName],
				canActivateMiddleware,
			)
		}
	}

	// REST module guards
	for _, moduleGuard := range app.module.RESTGuards {

		canActivateMiddleware := func(canActiveFn common.CanActivate) context.Handler {
			return func(ctx *context.Context) {
				common.HandleGuard(ctx, canActiveFn(ctx))
			}
		}(moduleGuard.Handler.(common.CanActivate))

		app.route.For(moduleGuard.Route, []string{moduleGuard.Method})(canActivateMiddleware)
	}

	// WS module guards
	for _, moduleGuard := range app.module.WSGuards {

		canActivateMiddleware := func(canActiveFn common.CanActivate) context.Handler {
			return func(ctx *context.Context) {
				common.HandleGuard(ctx, canActiveFn(ctx))
			}
		}(moduleGuard.Handler.(common.CanActivate))

		app.wsEventMap[moduleGuard.EventName] = append(
			app.wsEventMap[moduleGuard.EventName],
			canActivateMiddleware,
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
			aggregationInstance := aggregation.NewAggregation()
			endpoint := routing.ToEndpoint(routing.AddMethodToRoute(mainHandlerItem.Route, mainHandlerItem.Method))
			app.restAggregationMap[endpoint] = append(app.restAggregationMap[endpoint], aggregationInstance)
			interceptMiddleware := func(interceptor common.Interceptable, aggregationInstance *aggregation.Aggregation) context.Handler {
				return func(ctx *context.Context) {

					// IsMainHandlerCalled will be = true
					// if Pipe was invoked in Intercept function
					aggregationInstance.IsMainHandlerCalled = false
					aggregationInstance.SetMainData(nil)

					// invoke intercept function
					// value may returned from Pipe function
					// depend on Intercept invoked at run time
					value := interceptor.Intercept(ctx, aggregationInstance)
					aggregationInstance.InterceptorData = value
					app.setErrorAggregationOperators(ctx, aggregationInstance)

					ctx.Next()
				}
			}(globalInterceptor, aggregationInstance)

			app.route.For(mainHandlerItem.Route, []string{mainHandlerItem.Method})(interceptMiddleware)
		}

		// WS global interceptors
		for eventName := range insertedEvents {
			aggregationInstance := aggregation.NewAggregation()
			app.wsAggregationMap[eventName] = append(app.wsAggregationMap[eventName], aggregationInstance)
			interceptMiddleware := func(interceptor common.Interceptable, aggregationInstance *aggregation.Aggregation) context.Handler {
				return func(ctx *context.Context) {

					// IsMainHandlerCalled will be = true
					// if Pipe was invoked in Intercept function
					aggregationInstance.IsMainHandlerCalled = false
					aggregationInstance.SetMainData(nil)

					// invoke intercept function
					// value may returned from Pipe function
					// depend on Intercept invoked at run time
					value := interceptor.Intercept(ctx, aggregationInstance)
					aggregationInstance.InterceptorData = value
					app.setErrorAggregationOperators(ctx, aggregationInstance)

					ctx.Next()
				}
			}(globalInterceptor, aggregationInstance)

			app.wsEventMap[eventName] = append(
				app.wsEventMap[eventName],
				interceptMiddleware,
			)
		}
	}

	// REST module interceptors
	for _, moduleInterceptor := range app.module.RESTInterceptors {
		aggregationInstance := aggregation.NewAggregation()
		endpoint := routing.ToEndpoint(routing.AddMethodToRoute(moduleInterceptor.Route, moduleInterceptor.Method))
		app.restAggregationMap[endpoint] = append(app.restAggregationMap[endpoint], aggregationInstance)

		interceptMiddleware := func(interceptFn common.Intercept, aggregationInstance *aggregation.Aggregation) context.Handler {
			return func(ctx *context.Context) {

				// IsMainHandlerCalled will be = true
				// if Pipe was invoked in Intercept function
				aggregationInstance.IsMainHandlerCalled = false
				aggregationInstance.SetMainData(nil)

				// invoke intercept function
				// value may returned from Pipe function
				// depend on Intercept invoked at run time
				value := interceptFn(ctx, aggregationInstance)
				aggregationInstance.InterceptorData = value
				app.setErrorAggregationOperators(ctx, aggregationInstance)

				ctx.Next()
			}
		}(moduleInterceptor.Handler.(common.Intercept), aggregationInstance)

		// add interceptor middleware
		app.route.For(moduleInterceptor.Route, []string{moduleInterceptor.Method})(interceptMiddleware)
	}

	// WS module interceptors
	for _, moduleInterceptor := range app.module.WSInterceptors {
		aggregationInstance := aggregation.NewAggregation()
		app.wsAggregationMap[moduleInterceptor.EventName] = append(app.wsAggregationMap[moduleInterceptor.EventName], aggregationInstance)

		interceptMiddleware := func(interceptFn common.Intercept, aggregationInstance *aggregation.Aggregation) context.Handler {
			return func(ctx *context.Context) {

				// IsMainHandlerCalled will be = true
				// if Pipe was invoked in Intercept function
				aggregationInstance.IsMainHandlerCalled = false
				aggregationInstance.SetMainData(nil)

				// invoke intercept function
				// value may returned from Pipe function
				// depend on Intercept invoked at run time
				value := interceptFn(ctx, aggregationInstance)
				aggregationInstance.InterceptorData = value
				app.setErrorAggregationOperators(ctx, aggregationInstance)

				ctx.Next()
			}
		}(moduleInterceptor.Handler.(common.Intercept), aggregationInstance)

		app.wsEventMap[moduleInterceptor.EventName] = append(
			app.wsEventMap[moduleInterceptor.EventName],
			interceptMiddleware,
		)
	}

	// main REST handler
	for _, moduleHandler := range app.module.RESTMainHandlers {
		app.route.AddInjectableHandler(moduleHandler.Route, moduleHandler.Method, moduleHandler.Handler)
	}

	// main WS handler
	for _, moduleHandler := range app.module.WSMainHandlers {
		app.wsMainHandlerMap[moduleHandler.EventName] = moduleHandler.Handler
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

func (app *App) Use(handlers ...context.Handler) *App {
	for _, handler := range handlers {
		middleware := globalMiddleware{
			route:   "*",
			handler: handler,
		}
		app.globalMiddlewares = append(app.globalMiddlewares, middleware)
	}

	return app
}

func (app *App) UseLogger(logger common.Logger) *App {
	app.Logger = logger
	globalInterfaces[injectableInterfaces[0]] = app.Logger

	return app
}

func (app *App) For(route string) func(handlers ...context.Handler) *App {
	return func(handlers ...context.Handler) *App {
		for _, handler := range handlers {
			middleware := globalMiddleware{
				route:   route,
				handler: handler,
			}
			app.globalMiddlewares = append(app.globalMiddlewares, middleware)
		}

		return app
	}
}

func (app *App) Get(p Provider) any {
	k := genProviderKey(p)
	return utils.ArrFind(app.module.providers, func(provider Provider, i int) bool {
		return genProviderKey(provider) == k
	})
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := app.ctxPool.Get().(*context.Context)
	c.Timestamp = time.Now()
	c.ResponseWriter = w
	c.Request = r
	ctxID := app.getContextID(c)
	c.SetID(ctxID)

	defer app.ctxPool.Put(c)

	if utils.ArrIncludes[string](wsPaths, r.URL.Path) {
		c.SetType(context.WSType)
		websocket.Handler.ServeHTTP(func(wsConn *websocket.Conn) {
			app.handleWSRequest(wsConn, w, r, c)
		}, w, r)
	} else {
		c.SetType(context.HTTPType)
		c.ResponseWriter.Header().Set("X-Request-ID", c.GetID())

		app.handleRESTRequest(c)
	}

	c.Reset()
}

func (app *App) Listen(port int) error {
	app.route.Range(func(method, route string) {
		if route == "" {
			route = "/"
		}
		app.Logger.Info(
			"RouteExplorer",
			"method", method,
			"route", route,
		)
	})

	for eventName := range insertedEvents {
		p, e := context.ResolveWSEventname(eventName)
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

func (app *App) handleRESTRequest(c *context.Context) {
	var catchEvent string

	defer func() {
		if rec := recover(); rec != nil {
			if _, ok := app.catchRESTFnsMap[catchEvent]; ok {

				// Pipe errors run first
				// then exception filter
				if errorAggregationOperators, ok := c.Request.Context().Value(WithValueKey("ErrorAggregationOperators")).([]aggregation.AggregationOperator); ok {
					totalErrorAggregations := len(errorAggregationOperators)

					for i := totalErrorAggregations - 1; i >= 0; i-- {
						aggregation := errorAggregationOperators[i]
						rec = aggregation(c, rec)
					}
				}

				// 3rd param is index of catch function
				c.Event.Emit(catchEvent, c, rec, 0)
			}
		}
	}()

	isNext := true
	c.Next = func() {
		isNext = true
	}

	isMatched, matchedRoute, paramKeys, paramValues, handlers := app.route.Match(c.Request.URL.Path, c.Request.Method)
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

					if aggregations, ok := app.restAggregationMap[matchedRoute]; ok {
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
								returnREST(c, reflect.ValueOf(aggregation.InterceptorData))
								break
							}
						}

						if isMainHandlerCalled {
							returnREST(c, reflect.ValueOf(aggregatedData))
						}
					} else {
						if len(data) == 1 {
							returnREST(c, data[0])
						} else if len(data) > 1 {
							setStatusCode(c, data[0])
							returnREST(c, data[1])
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
			notFoundException := exception.NotFoundException(fmt.Sprintf("Cannot %v %v", c.Method, c.URL.Path))
			httpCode, _ := notFoundException.GetHTTPStatus()
			c.Status(httpCode)
			c.JSON(context.Map{
				"code":    notFoundException.GetCode(),
				"error":   notFoundException.Error(),
				"message": notFoundException.GetResponse(),
			})
		}
	}
}

func (app *App) handleWSRequest(wsConn *websocket.Conn, w http.ResponseWriter, r *http.Request, c *context.Context) {
	wsInstance := context.NewWS(wsConn)
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

	if !wsInstance.CanEstablish(insertedEvents) {
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

		var wsMsg context.WSMessage
		err = json.Unmarshal(message, &wsMsg)
		if err != nil {
			app.wsInvokeMiddlewares(c, exception.UnsupportedMediaTypeException(err.Error()))
			continue
		}

		// event was registered by controller
		var publishEventName string
		defer func() {
			if rec := recover(); rec != nil {
				if _, ok := app.catchWSFnsMap[publishEventName]; ok {

					// Pipe errors run first
					// then exception filter
					if errorAggregationOperators, ok := c.Request.Context().Value(WithValueKey("ErrorAggregationOperators")).([]aggregation.AggregationOperator); ok {
						totalErrorAggregations := len(errorAggregationOperators)

						for i := totalErrorAggregations - 1; i >= 0; i-- {
							aggregation := errorAggregationOperators[i]
							rec = aggregation(c, rec)
						}
					}

					// 3rd param is index of catch function
					c.Event.Emit(publishEventName, c, rec, 0)
				}

				// reset ErrorAggregationOperators
				// to prevent duplicate error aggregation
				// due to error will be added
				// whenever interceptor triggered
				// but WS 1 connection use 1 ctx
				newCtx := stdContext.WithValue(c.Request.Context(), WithValueKey("ErrorAggregationOperators"), nil)
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

						if aggregations, ok := app.wsAggregationMap[publishEventName]; ok {
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
									app.publishWSEvent(configPublishedEventName, wsid, wsMsg, c)
									break
								}
							}

							if isMainHandlerCalled {
								wsMsg := toWSMessage(reflect.ValueOf(aggregatedData))
								app.publishWSEvent(configPublishedEventName, wsid, wsMsg, c)
							}
						} else {
							if len(data) > 1 {
								wsMsg := toWSMessage(data[1])
								app.publishWSEvent(configPublishedEventName, wsid, wsMsg, c)
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

func (app *App) provideAndInvoke(f any, c *context.Context) []reflect.Value {
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

func (app *App) addWSEvent(subscribedEventName, wsid string, c *context.Context, cb func(args ...any)) {

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

func (app *App) removeWSEvent(subscribedEventName, wsid string, c *context.Context) {
	c.Event.RemoveAllListeners(subscribedEventName + wsid)
	if wsids, ok := app.wsEventToID.Load(subscribedEventName); ok {
		wsids = utils.ArrFilter[string](wsids.([]string), func(el string, i int) bool {
			return el != wsid
		})
		app.wsEventToID.Swap(subscribedEventName, wsids)
	}
}

func (app *App) publishWSEvent(configPublishedEventName, wsid, wsMsg string, c *context.Context) {
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
	newCtx := stdContext.WithValue(c.Request.Context(), WithValueKey("ErrorAggregationOperators"), nil)
	c.Request = c.Request.WithContext(newCtx)
}

func (app *App) wsInvokeMiddlewares(c *context.Context, exception exception.HTTPException) {
	isNext := true
	c.Next = func() {
		isNext = true
	}

	for _, globalMiddleware := range app.globalMiddlewares {
		if globalMiddleware.route == "*" && isNext {
			isNext = false
			globalMiddleware.handler(c)
		}
	}

	if isNext {
		c.WS.SendSelf(c, context.Map{
			"code":    exception.GetCode(),
			"error":   exception.Error(),
			"message": exception.GetResponse(),
		})
	}
}

func (app *App) getContextID(c *context.Context) string {
	reqID := c.Header().Get("X-Request-ID")
	if reqID == "" {
		uuid, _ := utils.StrUUID()
		return uuid
	}

	return reqID
}

func (app *App) setErrorAggregationOperators(ctx *context.Context, aggregationInstance *aggregation.Aggregation) {
	errorAggregationOpr := aggregationInstance.GetAggregationOperator(aggregation.OPERATOR_ERROR)
	if errorAggregationOpr != nil {
		errorAggregationOperators := ctx.Request.Context().Value(WithValueKey("ErrorAggregationOperators"))
		if errorAggregationOperators == nil {
			errorAggregationOperators = []aggregation.AggregationOperator{}
		}
		errorAggregationOperators = append(errorAggregationOperators.([]aggregation.AggregationOperator), errorAggregationOpr)

		newCtx := stdContext.WithValue(ctx.Request.Context(), WithValueKey("ErrorAggregationOperators"), errorAggregationOperators)
		ctx.Request = ctx.Request.WithContext(newCtx)
	}
}
