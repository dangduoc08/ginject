package core

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/dangduoc08/gooh/common"
	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
	"github.com/dangduoc08/gooh/utils"
)

type globalMiddleware struct {
	route   string
	handler context.Handler
}

type App struct {
	route              *routing.Router
	module             *Module
	pool               sync.Pool
	globalMiddlewares  []globalMiddleware
	globalGuarders     []common.Guarder
	globalInterceptors []common.Interceptable
	injectedProviders  map[string]Provider
}

// link to aliases
const (
	CONTEXT  = "/*context.Context"
	REQUEST  = "/*http.Request"
	RESPONSE = "net/http/http.ResponseWriter"
	PARAM    = "github.com/dangduoc08/gooh/context/context.Values"
	QUERY    = "net/url/url.Values"
	HEADER   = "net/http/http.Header"
	NEXT     = "/func()"
	REDIRECT = "/func(string)"
	PIPEABLE = "PIPEABLE"
)

var dependencies = map[string]int{
	CONTEXT:  1,
	REQUEST:  1,
	RESPONSE: 1,
	PARAM:    1,
	QUERY:    1,
	HEADER:   1,
	NEXT:     1,
	REDIRECT: 1,
	PIPEABLE: 1,
}

func New() *App {
	event := context.NewEvent()

	app := App{
		route: routing.NewRouter(),
		pool: sync.Pool{
			New: func() any {
				c := context.NewContext()
				c.Event = event

				return c
			},
		},
	}

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

		canActivateMiddleware := func(ctx *context.Context) {
			common.HandleGuard(ctx, globalGuard.CanActivate(ctx))
		}

		for _, mainHandlerItem := range app.module.MainHandlers {
			app.route.For(mainHandlerItem.Route, []string{mainHandlerItem.Method})(canActivateMiddleware)
		}
	}

	// module guards
	for _, moduleGuard := range app.module.Guards {
		canActivateMiddleware := func(ctx *context.Context) {
			common.HandleGuard(ctx, moduleGuard.Handler.(func(*context.Context) bool)(ctx))
		}
		app.route.For(moduleGuard.Route, []string{moduleGuard.Method})(canActivateMiddleware)
	}

	// global interceptors
	for _, globalInterceptor := range app.globalInterceptors {
		newGlobalInterceptor := injectDependencies(globalInterceptor, "interceptor", injectedProviders)
		globalInterceptor = common.Construct(newGlobalInterceptor.Interface(), "NewInterceptor").(common.Interceptable)

		interceptorMiddleware := func(ctx *context.Context) {
			globalInterceptor.Intercept(ctx, 1)

			ctx.Next()
		}

		for _, mainHandlerItem := range app.module.MainHandlers {
			app.route.For(mainHandlerItem.Route, []string{mainHandlerItem.Method})(interceptorMiddleware)
		}
	}

	// module interceptors
	for _, moduleInterceptor := range app.module.Interceptors {
		interceptorMiddleware := func(ctx *context.Context) {
			moduleInterceptor.Handler.(func(*context.Context, any) any)(ctx, 1)

			ctx.Next()
		}
		app.route.For(moduleInterceptor.Route, []string{moduleInterceptor.Method})(interceptorMiddleware)
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
	defer func() {
		if rec := recover(); rec != nil {
			c.Event.Emit(context.REQUEST_FAILED, c, rec)
		}
	}()

	isNext := true
	c.ResponseWriter = w
	c.Request = r
	c.Next = func() {
		isNext = true
	}

	isMatched, matchedRoute, paramKeys, paramValues, handlers := app.route.Match(r.URL.Path, r.Method)

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
					values := app.provideAndInvoke(injectableHandler, c)
					if len(values) == 1 {
						app.selectData(c, values[0])
					} else if len(values) > 1 {
						app.selectStatusCode(c, values[0])
						app.selectData(c, values[1])
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
			c.Status(http.StatusNotFound)
			c.Event.Emit(context.REQUEST_FINISHED, c)
			http.NotFound(w, r)
		}
	}
}

func (app *App) provideAndInvoke(f any, c *context.Context) []reflect.Value {
	args := []reflect.Value{}
	getFnArgs(f, func(dynamicArgKey string, i int, pipe reflect.Type) {
		if _, ok := dependencies[dynamicArgKey]; ok {
			args = append(args, reflect.ValueOf(app.getDependency(dynamicArgKey, c, pipe)))
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

func (app *App) getDependency(k string, c *context.Context, pipeType reflect.Type) any {
	switch k {
	case CONTEXT:
		return c
	case REQUEST:
		return c.Request
	case RESPONSE:
		return c.ResponseWriter
	case PARAM:
		return c.Param()
	case QUERY:
		return c.URL.Query()
	case HEADER:
		return c.Request.Header
	case NEXT:
		return c.Next
	case REDIRECT:
		return c.Redirect
	case PIPEABLE:
		var bindParam any = nil
		paramType := ""

		for i := 0; i < pipeType.NumField(); i++ {
			fieldKey := genFieldKey(pipeType.Field(i).Type)
			if fieldKey == QUERY {
				bindParam = c.URL.Query()
				paramType = "query"
			} else if fieldKey == PARAM {
				bindParam = c.Param()
				paramType = "param"
			} else if fieldKey == HEADER {
				bindParam = c.Request.Header
				paramType = "header"
			} else if app.injectedProviders[fieldKey] != nil {
				// injectDependencies(pipeType, "pipe", app.injectedProviders)
			}
		}

		return reflect.
			New(pipeType).
			Interface().(common.Pipeable).
			Transform(bindParam, common.ArgumentMetadata{
				ParamType: paramType,
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
