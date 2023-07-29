package core

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/dangduoc08/gooh/aggregation"
	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/exception"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type globalMiddleware struct {
	route   string
	handler context.Handler
}

type App struct {
	route                  *routing.Router
	module                 *Module
	pool                   sync.Pool
	globalMiddlewares      []globalMiddleware
	globalGuarders         []common.Guarder
	globalInterceptors     []common.Interceptable
	globalExceptionFilters []common.ExceptionFilterable
	injectedProviders      map[string]Provider
	aggregationMap         map[string][]*aggregation.Aggregation
	catchFnsMap            map[string][]common.Catch
}

// link to aliases
const (
	CONTEXT         = "/*context.Context"
	REQUEST         = "/*http.Request"
	RESPONSE        = "net/http/http.ResponseWriter"
	BODY            = "github.com/dangduoc08/gooh/context/context.Body"
	QUERY           = "net/url/url.Values"
	HEADER          = "net/http/http.Header"
	PARAM           = "github.com/dangduoc08/gooh/context/context.Param"
	NEXT            = "/func()"
	REDIRECT        = "/func(string)"
	BODY_PIPEABLE   = "body"
	QUERY_PIPEABLE  = "query"
	HEADER_PIPEABLE = "header"
	PARAM_PIPEABLE  = "param"
)

var dependencies = map[string]int{
	CONTEXT:         1,
	REQUEST:         1,
	RESPONSE:        1,
	BODY:            1,
	QUERY:           1,
	HEADER:          1,
	PARAM:           1,
	NEXT:            1,
	REDIRECT:        1,
	BODY_PIPEABLE:   1,
	QUERY_PIPEABLE:  1,
	HEADER_PIPEABLE: 1,
	PARAM_PIPEABLE:  1,
}

func New() *App {
	event := context.NewEvent()

	app := App{
		route:          routing.NewRouter(),
		aggregationMap: make(map[string][]*aggregation.Aggregation),
		catchFnsMap:    make(map[string][]common.Catch),
		pool: sync.Pool{
			New: func() any {
				c := context.NewContext()
				c.Event = event

				return c
			},
		},
	}

	// binding default exception filter
	app.BindGlobalExceptionFilters(GlobalExceptionFilter{})

	return &app
}

func (app *App) Create(m *Module) {
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

	// global middlewares
	for _, globalMiddleware := range app.globalMiddlewares {
		if globalMiddleware.route != "ALL" {
			app.route.For(globalMiddleware.route, routing.HTTPMethods)(globalMiddleware.handler)
		} else {
			app.route.Use(globalMiddleware.handler)
		}
	}

	// module middlewares
	for _, moduleMiddleware := range app.module.Middlewares {
		app.route.For(moduleMiddleware.Route, []string{moduleMiddleware.Method})(moduleMiddleware.Handlers...)
	}

	// global guards
	for _, globalGuard := range app.globalGuarders {
		newGlobalGuard := injectDependencies(globalGuard, "guard", injectedProviders)
		globalGuard = common.Construct(newGlobalGuard.Interface(), "NewGuard").(common.Guarder)

		canActivateMiddleware := func(guard common.Guarder) context.Handler {
			return func(ctx *context.Context) {
				common.HandleGuard(ctx, guard.CanActivate(ctx))
			}
		}(globalGuard)

		for _, mainHandlerItem := range app.module.MainHandlers {
			app.route.For(mainHandlerItem.Route, []string{mainHandlerItem.Method})(canActivateMiddleware)
		}
	}

	// module guards
	for _, moduleGuard := range app.module.Guards {

		canActivateMiddleware := func(canActiveFn common.CanActivate) context.Handler {
			return func(ctx *context.Context) {
				common.HandleGuard(ctx, canActiveFn(ctx))
			}
		}(moduleGuard.Handler.(common.CanActivate))

		app.route.For(moduleGuard.Route, []string{moduleGuard.Method})(canActivateMiddleware)
	}

	// global interceptors
	for _, globalInterceptor := range app.globalInterceptors {
		newGlobalInterceptor := injectDependencies(globalInterceptor, "interceptor", injectedProviders)
		globalInterceptor = common.Construct(newGlobalInterceptor.Interface(), "NewInterceptor").(common.Interceptable)

		for _, mainHandlerItem := range app.module.MainHandlers {
			aggregationInstance := aggregation.NewAggregation()
			endpoint := routing.ToEndpoint(routing.AddMethodToRoute(mainHandlerItem.Route, mainHandlerItem.Method))
			app.aggregationMap[endpoint] = append(app.aggregationMap[endpoint], aggregationInstance)
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

					ctx.Next()
				}
			}(globalInterceptor, aggregationInstance)

			app.route.For(mainHandlerItem.Route, []string{mainHandlerItem.Method})(interceptMiddleware)
		}
	}

	for _, moduleInterceptor := range app.module.Interceptors {
		aggregationInstance := aggregation.NewAggregation()
		endpoint := routing.ToEndpoint(routing.AddMethodToRoute(moduleInterceptor.Route, moduleInterceptor.Method))
		app.aggregationMap[endpoint] = append(app.aggregationMap[endpoint], aggregationInstance)

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

				ctx.Next()
			}
		}(moduleInterceptor.Handler.(common.Intercept), aggregationInstance)

		// add interceptor middleware
		app.route.For(moduleInterceptor.Route, []string{moduleInterceptor.Method})(interceptMiddleware)
	}

	// module exception filters
	totalModuleExceptionFilers := len(app.module.ExceptionFilters)
	for i := totalModuleExceptionFilers - 1; i >= 0; i-- {
		moduleExceptionFilter := app.module.ExceptionFilters[i]
		endpoint := routing.ToEndpoint(routing.AddMethodToRoute(moduleExceptionFilter.Route, moduleExceptionFilter.Method))
		app.catchFnsMap[endpoint] = append(app.catchFnsMap[endpoint], moduleExceptionFilter.Handler.(common.Catch))
	}

	// global exception filters
	totalGlobalExceptionFilters := len(app.globalExceptionFilters)
	for i := totalGlobalExceptionFilters - 1; i >= 0; i-- {
		globalExceptionFilter := app.globalExceptionFilters[i]
		newGlobalExceptionFilter := injectDependencies(globalExceptionFilter, "exceptionFilter", injectedProviders)
		globalExceptionFilter = common.Construct(newGlobalExceptionFilter.Interface(), "NewExceptionFilter").(common.ExceptionFilterable)

		for _, mainHandlerItem := range app.module.MainHandlers {
			endpoint := routing.ToEndpoint(routing.AddMethodToRoute(mainHandlerItem.Route, mainHandlerItem.Method))
			app.catchFnsMap[endpoint] = append(app.catchFnsMap[endpoint], globalExceptionFilter.Catch)
		}
	}

	for pattern, catchFns := range app.catchFnsMap {
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

	// main handler
	for _, moduleHandler := range app.module.MainHandlers {
		app.route.AddInjectableHandler(moduleHandler.Route, moduleHandler.Method, moduleHandler.Handler)
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
			route:   "ALL",
			handler: handler,
		}
		app.globalMiddlewares = append(app.globalMiddlewares, middleware)
	}

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
	c := app.pool.Get().(*context.Context)
	c.Timestamp = time.Now()
	defer app.pool.Put(c)
	app.handleRequest(w, r, c)
	c.Reset()
}

func (app *App) ListenAndServe(addr string) error {
	app.route.Range(func(method, route string) {
		log.Default().Println("RouteExplorer", method, route)
	})

	return http.ListenAndServe(addr, app)
}

func (app *App) handleRequest(w http.ResponseWriter, r *http.Request, c *context.Context) {
	var catchEvent string
	defer func() {
		if rec := recover(); rec != nil {
			if _, ok := app.catchFnsMap[catchEvent]; ok {
				// 3rd param is index of catch function
				c.Event.Emit(catchEvent, c, rec, 0)
			}
		}
	}()

	isNext := true
	c.ResponseWriter = w
	c.Request = r
	c.Next = func() {
		isNext = true
	}

	isMatched, matchedRoute, paramKeys, paramValues, handlers := app.route.Match(r.URL.Path, r.Method)
	catchEvent = matchedRoute

	if isMatched {
		c.SetRoute(matchedRoute)
		c.ParamKeys = paramKeys
		c.ParamValues = paramValues
		if r.Method == http.MethodPost {
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

					if aggregations, ok := app.aggregationMap[matchedRoute]; ok {
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
										app.selectStatusCode(c, data[0])
										aggregatedData = data[1].Interface()
									}
								}

								aggregation.SetMainData(aggregatedData)
								aggregatedData = aggregation.Aggregate()
							} else {
								isMainHandlerCalled = false
								app.selectData(c, reflect.ValueOf(aggregation.InterceptorData))
								break
							}
						}

						if isMainHandlerCalled {
							app.selectData(c, reflect.ValueOf(aggregatedData))
						}
					} else {
						if len(data) == 1 {
							app.selectData(c, data[0])
						} else if len(data) > 1 {
							app.selectStatusCode(c, data[0])
							app.selectData(c, data[1])
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
			c.Event.Emit(context.REQUEST_FINISHED, c)
			c.JSON(context.Map{
				"code":    notFoundException.GetCode(),
				"error":   notFoundException.Error(),
				"message": notFoundException.GetResponse(),
			})
		}
	}
}

func (app *App) provideAndInvoke(f any, c *context.Context) []reflect.Value {
	args := []reflect.Value{}
	getFnArgs(f, app.injectedProviders, func(dynamicArgKey string, i int, pipeValue reflect.Value) {
		if _, ok := dependencies[dynamicArgKey]; ok {
			args = append(args, reflect.ValueOf(app.getDependency(dynamicArgKey, c, pipeValue)))
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

func (app *App) getDependency(k string, c *context.Context, pipeValue reflect.Value) any {
	switch k {
	case CONTEXT:
		return c
	case REQUEST:
		return c.Request
	case RESPONSE:
		return c.ResponseWriter
	case BODY:
		return c.Body()
	case QUERY:
		return c.Query()
	case HEADER:
		return c.Header()
	case PARAM:
		return c.Param()
	case NEXT:
		return c.Next
	case REDIRECT:
		return c.Redirect
	case BODY_PIPEABLE:
		return pipeValue.
			Interface().(common.BodyPipeable).
			Transform(c.Body(), common.ArgumentMetadata{
				ParamType: BODY_PIPEABLE,
			})
	case QUERY_PIPEABLE:
		return pipeValue.
			Interface().(common.QueryPipeable).
			Transform(c.Query(), common.ArgumentMetadata{
				ParamType: QUERY_PIPEABLE,
			})
	case HEADER_PIPEABLE:
		return pipeValue.
			Interface().(common.HeaderPipeable).
			Transform(c.Header(), common.ArgumentMetadata{
				ParamType: HEADER_PIPEABLE,
			})
	case PARAM_PIPEABLE:
		return pipeValue.
			Interface().(common.ParamPipeable).
			Transform(c.Param(), common.ArgumentMetadata{
				ParamType: PARAM_PIPEABLE,
			})
	}

	return dependencies
}

func (app *App) selectData(c *context.Context, data reflect.Value) {
	switch data.Type().Kind() {
	case
		reflect.Map,
		reflect.Slice,
		reflect.Struct,
		reflect.Interface:
		c.JSON(data.Interface())
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		c.Text(fmt.Sprint(data))
	case
		reflect.Pointer,
		reflect.UnsafePointer:
		c.Text(fmt.Sprint(data.UnsafePointer()))
	case
		reflect.String:
		c.Text(data.Interface().(string))
	case
		reflect.Func:
		c.Text(data.Type().String())
	}
}

func (app *App) selectStatusCode(c *context.Context, statusCode reflect.Value) {
	statusCodeKind := statusCode.Type().Kind()

	if statusCodeKind == reflect.Int {
		status := int(statusCode.Int())
		if http.StatusText(status) != "" {
			c.Status(status)
		}
	} else if statusCodeKind == reflect.Interface {
		if status, ok := statusCode.Interface().(int); ok &&
			http.StatusText(status) != "" {
			c.Status(status)
		}
	}
}
