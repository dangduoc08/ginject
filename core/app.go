package core

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/devtool"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/versioning"
	"golang.org/x/net/websocket"
)

type App struct {
	http    *HTTP
	ws      *WS
	ctxPool sync.Pool

	isEnableDevtool bool
	devtool         *devtool.Devtool

	module *Module

	globalMiddlewares      []common.MiddlewareFn
	globalGuarders         []common.Guarder
	globalInterceptors     []common.Interceptable
	globalExceptionFilters []common.ExceptionFilterable
	injectedProviders      map[string]Provider

	Logger common.Logger
}

// Reflect-type keys mirroring the public type aliases declared in aliases.go;
// used to resolve handler parameters by their type during injection.
const (
	contextKey      = "/*ctx.Context"
	wsConnectionKey = "/*websocket.Conn"
	requestKey      = "/*http.Request"
	responseKey     = "net/http/http.ResponseWriter"
	bodyKey         = "github.com/dangduoc08/ginject/ctx/ctx.Body"
	formKey         = "github.com/dangduoc08/ginject/ctx/ctx.Form"
	queryKey        = "github.com/dangduoc08/ginject/ctx/ctx.Query"
	headerKey       = "github.com/dangduoc08/ginject/ctx/ctx.Header"
	paramKey        = "github.com/dangduoc08/ginject/ctx/ctx.Param"
	fileKey         = "github.com/dangduoc08/ginject/ctx/ctx.File"
	wsPayloadKey    = "github.com/dangduoc08/ginject/ctx/ctx.WSPayload"
	nextKey         = "/func()"
	redirectKey     = "/func(string)"
)

var dependencies = map[string]int{
	contextKey:                  1,
	wsConnectionKey:             1,
	requestKey:                  1,
	responseKey:                 1,
	bodyKey:                     1,
	formKey:                     1,
	queryKey:                    1,
	headerKey:                   1,
	paramKey:                    1,
	fileKey:                     1,
	wsPayloadKey:                1,
	nextKey:                     1,
	redirectKey:                 1,
	common.ContextPipeableKey:   1,
	common.BodyPipeableKey:      1,
	common.FormPipeableKey:      1,
	common.QueryPipeableKey:     1,
	common.HeaderPipeableKey:    1,
	common.ParamPipeableKey:     1,
	common.FilePipeableKey:      1,
	common.WSPayloadPipeableKey: 1,
}

type WithValueKey string

type corsOriginChecker interface {
	AllowedOrigin(string) bool
}

func New() *App {
	app := &App{
		http: newHTTP(),
		ws:   newWS(),
		ctxPool: sync.Pool{
			New: func() any {
				c := ctx.NewContext()
				c.Broker = broker.NewWithConfig(
					broker.Config{
						RecoverPanics: true,
					},
				)
				return c
			},
		},
	}

	app.BindGlobalExceptionFilters(globalExceptionFilter{})

	return app
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := app.ctxPool.Get().(*ctx.Context)
	c.Init(w, r)

	if r.URL.Path == "/ws" || r.URL.Path == "/ws/" {
		c.SetType(ctx.WSType)
		app.ws.upgrade(w, r, websocket.Server{
			Handshake: func(cfg *websocket.Config, r *http.Request) error {
				defer app.releaseCtx(c)
				return app.ws.handshake(cfg, r)
			},
			Handler: websocket.Handler(app.ws.handleRequest),
		})

		return
	}

	defer app.releaseCtx(c)
	c.SetType(ctx.HTTPType)
	c.ResponseWriter.Header().Set(ctx.RequestID, c.GetID())
	app.http.handleRequest(c)
}

func (app *App) Create(m *Module) {
	app.initLogger()
	injectedProviders := app.initProviders(m)
	app.initExceptionFilters(injectedProviders)
	app.bindCatchMiddlewares()
	app.initMiddlewares(injectedProviders)
	app.initGuards(injectedProviders)
	app.initInterceptors(injectedProviders)
	app.initMainHandlers()
	app.ws.buildCompiledPatterns()
	if app.isEnableDevtool {
		app.createDevtool()
	}
}

func (app *App) initLogger() {
	if app.Logger == nil {
		app.Logger = log.NewLog(nil)
	}
	globalInterfaces[injectableInterfaces[0]] = app.Logger
}

func (app *App) initProviders(m *Module) map[string]Provider {
	app.module = m.NewModule()

	injectedProviders := make(map[string]Provider, len(app.module.providers))
	for _, provider := range app.module.providers {
		injectedProviders[genProviderKey(provider)] = provider
	}
	app.injectedProviders = injectedProviders

	invokeHandler := func(f any, c *ctx.Context) []reflect.Value {
		return invokeHandlerByProviders(f, injectedProviders, c)
	}
	app.http.invokeHandler = invokeHandler
	app.ws.invokeHandler = invokeHandler
	app.ws.globalMiddlewares = &app.globalMiddlewares

	return injectedProviders
}

func (app *App) initExceptionFilters(injectedProviders map[string]Provider) {
	for i := len(app.module.RESTExceptionFilters) - 1; i >= 0; i-- {
		ef := app.module.RESTExceptionFilters[i]
		httpMethod := routing.OperationsMapHTTPMethods[ef.Method]
		endpoint := routing.MethodRouteVersionToPattern(httpMethod, ef.Route, ef.Version)
		app.http.catchFnsMap[endpoint] = append(app.http.catchFnsMap[endpoint], ef.Handler.(common.Catch))
	}

	for i := len(app.module.WSExceptionFilters) - 1; i >= 0; i-- {
		ef := app.module.WSExceptionFilters[i]
		app.ws.catchFnsMap[ef.EventName] = append(app.ws.catchFnsMap[ef.EventName], ef.Handler.(common.Catch))
	}

	if len(app.globalExceptionFilters) > 0 {
		wsEventNames := getWSEventKeys()
		for i := len(app.globalExceptionFilters) - 1; i >= 0; i-- {
			gef := app.globalExceptionFilters[i]
			newGef, err := injectDependencies(gef, "exceptionFilter", injectedProviders)
			if err != nil {
				panic(err)
			}
			gef = common.Construct(newGef.Interface(), "NewExceptionFilter").(common.ExceptionFilterable)

			for _, h := range app.module.RESTMainHandlers {
				httpMethod := routing.OperationsMapHTTPMethods[h.Method]
				endpoint := routing.MethodRouteVersionToPattern(httpMethod, h.Route, h.Version)
				app.http.catchFnsMap[endpoint] = append(app.http.catchFnsMap[endpoint], gef.Catch)
			}
			for _, eventName := range wsEventNames {
				app.ws.catchFnsMap[eventName] = append(app.ws.catchFnsMap[eventName], gef.Catch)
			}
		}
	}
}

func (app *App) bindCatchMiddlewares() {
	for pattern, catchFns := range app.http.catchFnsMap {
		mw := buildCatchMiddleware(pattern, catchFns)
		method, route, version := routing.PatternToMethodRouteVersion(pattern)
		httpMethod := routing.OperationsMapHTTPMethods[method]
		app.http.route.For([]string{httpMethod}, route, version)(mw)
	}
	for pattern, catchFns := range app.ws.catchFnsMap {
		mw := buildCatchMiddleware(pattern, catchFns)
		app.ws.eventMap[pattern] = append(app.ws.eventMap[pattern], mw)
	}
}

func (app *App) initMiddlewares(injectedProviders map[string]Provider) {
	if len(app.globalMiddlewares) > 0 {
		wsEventNames := getWSEventKeys()
		for _, gm := range app.globalMiddlewares {
			newGM, err := injectDependencies(gm, "middleware", injectedProviders)
			if err != nil {
				panic(err)
			}
			gm = common.Construct(newGM.Interface(), "NewMiddleware").(common.MiddlewareFn)
			if app.ws.corsAllowOrigin == nil {
				if checker, ok := gm.(corsOriginChecker); ok {
					app.ws.corsAllowOrigin = checker.AllowedOrigin
				}
			}
			mw := func(middleware common.MiddlewareFn) ctx.Handler {
				return func(c *ctx.Context) { middleware.Use(c, c.Next) }
			}(gm)
			app.http.route.Use(mw)
			for _, eventName := range wsEventNames {
				app.ws.eventMap[eventName] = append(app.ws.eventMap[eventName], mw)
			}
		}
	}

	for _, rm := range app.module.RESTMiddlewares {
		mw := buildUseMiddleware(rm.Handler.(common.Use))
		httpMethod := routing.OperationsMapHTTPMethods[rm.Method]
		app.http.route.For([]string{httpMethod}, rm.Route, rm.Version)(mw)
	}

	for _, wm := range app.module.WSMiddlewares {
		mw := buildUseMiddleware(wm.Handler.(common.Use))
		app.ws.eventMap[wm.EventName] = append(app.ws.eventMap[wm.EventName], mw)
	}
}

func (app *App) initGuards(injectedProviders map[string]Provider) {
	if len(app.globalGuarders) > 0 {
		wsEventNames := getWSEventKeys()
		for _, gg := range app.globalGuarders {
			newGG, err := injectDependencies(gg, "guard", injectedProviders)
			if err != nil {
				panic(err)
			}
			gg = common.Construct(newGG.Interface(), "NewGuard").(common.Guarder)
			mw := func(guard common.Guarder) ctx.Handler {
				return func(c *ctx.Context) { common.HandleGuard(c, guard.CanActivate(c)) }
			}(gg)
			for _, h := range app.module.RESTMainHandlers {
				httpMethod := routing.OperationsMapHTTPMethods[h.Method]
				app.http.route.For([]string{httpMethod}, h.Route, h.Version)(mw)
			}
			for _, eventName := range wsEventNames {
				app.ws.eventMap[eventName] = append(app.ws.eventMap[eventName], mw)
			}
		}
	}

	for _, mg := range app.module.RESTGuards {
		mw := buildGuardMiddleware(mg.Handler.(common.CanActivate))
		httpMethod := routing.OperationsMapHTTPMethods[mg.Method]
		app.http.route.For([]string{httpMethod}, mg.Route, mg.Version)(mw)
	}

	for _, mg := range app.module.WSGuards {
		mw := buildGuardMiddleware(mg.Handler.(common.CanActivate))
		app.ws.eventMap[mg.EventName] = append(app.ws.eventMap[mg.EventName], mw)
	}
}

func (app *App) initInterceptors(injectedProviders map[string]Provider) {
	if len(app.globalInterceptors) > 0 {
		wsEventNames := getWSEventKeys()
		for _, gi := range app.globalInterceptors {
			newGI, err := injectDependencies(gi, "interceptor", injectedProviders)
			if err != nil {
				panic(err)
			}
			gi = common.Construct(newGI.Interface(), "NewInterceptor").(common.Interceptable)

			for _, h := range app.module.RESTMainHandlers {
				httpMethod := routing.OperationsMapHTTPMethods[h.Method]
				endpoint := routing.MethodRouteVersionToPattern(httpMethod, h.Route, h.Version)
				mw := buildInterceptMiddleware(endpoint, gi.Intercept)
				app.http.route.For([]string{httpMethod}, h.Route, h.Version)(mw)
			}
			for _, eventName := range wsEventNames {
				mw := buildInterceptMiddleware(eventName, gi.Intercept)
				app.ws.eventMap[eventName] = append(app.ws.eventMap[eventName], mw)
			}
		}
	}

	for _, mi := range app.module.RESTInterceptors {
		httpMethod := routing.OperationsMapHTTPMethods[mi.Method]
		endpoint := routing.MethodRouteVersionToPattern(httpMethod, mi.Route, mi.Version)
		mw := buildInterceptMiddleware(endpoint, mi.Handler.(common.Intercept))
		app.http.route.For([]string{httpMethod}, mi.Route, mi.Version)(mw)
	}

	for _, mi := range app.module.WSInterceptors {
		mw := buildInterceptMiddleware(mi.EventName, mi.Handler.(common.Intercept))
		app.ws.eventMap[mi.EventName] = append(app.ws.eventMap[mi.EventName], mw)
	}
}

func (app *App) initMainHandlers() {
	for _, h := range app.module.RESTMainHandlers {
		app.http.addMainHandler(h)
	}
	for _, h := range app.module.WSMainHandlers {
		app.ws.mainHandlerMap[h.EventName] = h.Handler
	}
}

func (app *App) releaseCtx(c *ctx.Context) {
	c.Reset()
	app.ctxPool.Put(c)
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
	app.http.enableVersioning(v)

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
	return app.injectedProviders[genProviderKey(p)]
}

func (app *App) Listen(port int) error {

	// REST logs
	var routeArr []string
	for _, items := range app.http.route.Hash {
		for _, item := range items {
			if item.HandlerIndex > -1 {
				routeArr = append(routeArr, item.Pattern)
			}
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
	eventArr := getWSEventKeys()
	sort.Strings(eventArr)

	for _, eventName := range eventArr {
		app.Logger.Info(
			"WebSocketEvent",
			"event", eventName,
		)
	}

	addr := fmt.Sprintf(":%v", port)

	server := &http.Server{
		Addr:    addr,
		Handler: app,

		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,

		MaxHeaderBytes: 1 << 20,
	}

	logBoostrap(port)

	return server.ListenAndServe()
}

func (app *App) createDevtool() {
	devtoolBuilder := devtool.DevtoolBuilder()

	app.devtool = devtoolBuilder.
		AddExceptionFilters(app.globalExceptionFilters, app.module.RESTExceptionFilters).
		AddMiddlewares(app.globalMiddlewares, app.module.RESTMiddlewares).
		AddGuarders(app.globalGuarders, app.module.RESTGuards).
		AddInterceptors(app.globalInterceptors, app.module.RESTInterceptors).
		AddVersioning(app.http.versioning).
		AddRESTMainHandlers(app.module.RESTMainHandlers).
		Build()

	go app.devtool.Serve()
}
