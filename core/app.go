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
	ctxPool sync.Pool

	isDevtoolEnabled bool
	devtool          *devtool.Devtool

	ws          *WS
	wsConfig    *WSConfig
	isWSEnabled bool

	module *Module

	globalMiddlewares      []common.MiddlewareFn
	globalGuarders         []common.Guarder
	globalInterceptors     []common.Interceptable
	globalExceptionFilters []common.ExceptionFilterable
	injectedProviders      map[string]Provider

	Logger common.Logger
}

const (
	contextKey      = "/*ctx.HTTPContext"
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

// knownDependencyKeys is the set of dependency-type keys the framework can
// resolve for a handler parameter (see getDependency); values are unused
// and always 1.
var knownDependencyKeys = map[string]int{
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

type WithValueKey = common.WithValueKey

func New() *App {
	app := &App{
		http: newHTTP(),
		ws:   nil,
		ctxPool: sync.Pool{
			New: func() any {
				c := ctx.NewHTTPContext()
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
	c := app.ctxPool.Get().(*ctx.HTTPContext)
	c.Init(w, r)

	if app.isWSEnabled && app.ws.isWSPath(r.URL.Path) {
		c.SetType(ctx.WSType)
		app.ws.upgrade(w, r, websocket.Server{
			Handshake: func(wsCfg *websocket.Config, r *http.Request) error {
				defer app.releaseCtx(c)

				c.Request = r
				c.SetWSConfig(wsCfg)
				return app.ws.handshake(c)
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

// Order matters here and is easy to break by accident:
//
//  1. initProviders must run first — it builds injectedProviders and
//     app.module (parsed from m), which everything below reads from.
//  2. initWS must run right after that, and BEFORE initExceptionFilters/
//     bindCatchMiddlewares/initGuards/initInterceptors/initMainHandlers —
//     those functions write straight into app.ws.catchFnsByEvent /
//     app.ws.eventMatcher whenever app.module has WS handlers, so app.ws
//     must already exist or it's a nil pointer dereference.
//  3. isDevtoolEnabled must run last — it reads the fully-populated
//     module/route state to build the devtool snapshot.
//
// Do not reorder these calls without re-checking every init* function for
// a app.ws.* access.
func (app *App) Create(m *Module) {
	app.initLogger()
	injectedProviders := app.initProviders(m)
	app.initWS(injectedProviders)
	app.initMiddlewares(injectedProviders)
	app.initExceptionFilters(injectedProviders)
	app.bindCatchMiddlewares()
	app.initGuards(injectedProviders)
	app.initInterceptors(injectedProviders)
	app.initMainHandlers()

	if app.isDevtoolEnabled {
		app.createDevtool()
	}
}

func (app *App) initWS(injectedProviders map[string]Provider) {
	if !app.isWSEnabled {
		return
	}

	app.wsConfig.injectedProviders = injectedProviders
	app.wsConfig.logger = app.Logger
	app.wsConfig.resolveAndCallHandler = app.http.resolveAndCallHandler
	app.wsConfig.newCtx = func() *ctx.WSContext { return app.ctxPool.Get().(*ctx.WSContext) }
	app.wsConfig.releaseCtx = app.releaseCtx
	app.ws = NewWS(app.wsConfig)
}

func (app *App) initLogger() {
	if app.Logger == nil {
		app.Logger = log.NewLog(nil)
	}
	globalInterfaceByKey[injectableInterfaces[0]] = app.Logger
}

func (app *App) initProviders(m *Module) map[string]Provider {
	app.module = m.NewModule()

	injectedProviders := make(map[string]Provider, len(app.module.providers))
	for _, provider := range app.module.providers {
		injectedProviders[genProviderKey(provider)] = provider
	}
	app.injectedProviders = injectedProviders

	resolveAndCallHandler := func(f any, c *ctx.HTTPContext) []reflect.Value {
		return invokeHandlerByProviders(f, injectedProviders, c)
	}
	app.http.resolveAndCallHandler = resolveAndCallHandler

	return injectedProviders
}

func (app *App) initExceptionFilters(injectedProviders map[string]Provider) {
	for i := len(app.module.RESTExceptionFilters) - 1; i >= 0; i-- {
		ef := app.module.RESTExceptionFilters[i]
		httpMethod := routing.OperationsMapHTTPMethods[ef.Method]
		endpoint := routing.MethodRouteVersionToPattern(httpMethod, ef.Route, ef.Version)
		app.http.catchFnsByRoute[endpoint] = append(app.http.catchFnsByRoute[endpoint], ef.Handler.(common.RESTCatch))
	}

	for i := len(app.module.WSExceptionFilters) - 1; i >= 0; i-- {
		ef := app.module.WSExceptionFilters[i]
		app.ws.catchFnsByEvent[ef.EventName] = append(app.ws.catchFnsByEvent[ef.EventName], ef.Handler.(common.WSCatch))
	}

	if len(app.globalExceptionFilters) > 0 {
		for i := len(app.globalExceptionFilters) - 1; i >= 0; i-- {
			gef := app.globalExceptionFilters[i]
			newGef, err := injectDependencies(gef, "exceptionFilter", injectedProviders)
			if err != nil {
				panic(err)
			}
			exceptionFilterable := common.Construct(newGef.Interface(), "NewExceptionFilter")

			restCatch, isRESTFilter := common.AsRESTExceptionFilter(exceptionFilterable)
			wsCatch, isWSFilter := common.AsWSExceptionFilter(exceptionFilterable)
			if !isRESTFilter && !isWSFilter {
				panic(common.ExceptionFilterShapeError(exceptionFilterable))
			}

			if isRESTFilter {
				for _, h := range app.module.RESTMainHandlers {
					httpMethod := routing.OperationsMapHTTPMethods[h.Method]
					endpoint := routing.MethodRouteVersionToPattern(httpMethod, h.Route, h.Version)
					app.http.catchFnsByRoute[endpoint] = append(app.http.catchFnsByRoute[endpoint], restCatch)
				}
			}

			if isWSFilter {
				for _, h := range app.module.WSMainHandlers {
					app.ws.catchFnsByEvent[h.EventName] = append(app.ws.catchFnsByEvent[h.EventName], wsCatch)
				}
			}
		}
	}
}

func (app *App) bindCatchMiddlewares() {
	for pattern, catchFns := range app.http.catchFnsByRoute {
		mw := common.BuildHTTPCatchMiddleware(pattern, catchFns)
		method, route, version := routing.PatternToMethodRouteVersion(pattern)
		httpMethod := routing.OperationsMapHTTPMethods[method]
		app.http.route.For([]string{httpMethod}, route, version)(mw)
	}

	if app.ws != nil {
		for pattern, catchFns := range app.ws.catchFnsByEvent {
			mw := common.BuildWSCatchMiddleware(pattern, catchFns)
			app.ws.eventMatcher.AddMiddlewares(pattern, mw)
		}
	}
}

func (app *App) initMiddlewares(injectedProviders map[string]Provider) {
	if len(app.globalMiddlewares) > 0 {
		for _, gm := range app.globalMiddlewares {
			newGM, err := injectDependencies(gm, "middleware", injectedProviders)
			if err != nil {
				panic(err)
			}
			gm = common.Construct(newGM.Interface(), "NewMiddleware").(common.MiddlewareFn)
			mw := buildUseMiddleware(gm.Use)
			app.http.route.Use(mw)
		}
	}

	for _, rm := range app.module.RESTMiddlewares {
		mw := buildUseMiddleware(rm.Handler.(common.Use))
		httpMethod := routing.OperationsMapHTTPMethods[rm.Method]
		app.http.route.For([]string{httpMethod}, rm.Route, rm.Version)(mw)
	}
}

func (app *App) initGuards(injectedProviders map[string]Provider) {
	if len(app.globalGuarders) > 0 {
		for _, gg := range app.globalGuarders {
			newGG, err := injectDependencies(gg, "guard", injectedProviders)
			if err != nil {
				panic(err)
			}
			guarder := common.Construct(newGG.Interface(), "NewGuard")

			restCanActivate, isRESTGuard := common.AsRESTGuard(guarder)
			wsCanActivate, isWSGuard := common.AsWSGuard(guarder)
			if !isRESTGuard && !isWSGuard {
				panic(common.GuardShapeError(guarder))
			}

			if isRESTGuard {
				mw := common.BuildHTTPGuardMiddleware(restCanActivate)
				for _, h := range app.module.RESTMainHandlers {
					httpMethod := routing.OperationsMapHTTPMethods[h.Method]
					app.http.route.For([]string{httpMethod}, h.Route, h.Version)(mw)
				}
			}

			if isWSGuard {
				mw := common.BuildWSGuardMiddleware(wsCanActivate)
				for _, h := range app.module.WSMainHandlers {
					app.ws.eventMatcher.AddMiddlewares(h.EventName, mw)
				}
			}
		}
	}

	for _, mg := range app.module.RESTGuards {
		mw := common.BuildHTTPGuardMiddleware(mg.Handler.(common.RESTCanActivate))
		httpMethod := routing.OperationsMapHTTPMethods[mg.Method]
		app.http.route.For([]string{httpMethod}, mg.Route, mg.Version)(mw)
	}

	for _, mg := range app.module.WSGuards {
		mw := common.BuildWSGuardMiddleware(mg.Handler.(common.WSCanActivate))
		app.ws.eventMatcher.AddMiddlewares(mg.EventName, mw)
	}
}

func (app *App) initInterceptors(injectedProviders map[string]Provider) {
	if len(app.globalInterceptors) > 0 {
		for _, gi := range app.globalInterceptors {
			newGI, err := injectDependencies(gi, "interceptor", injectedProviders)
			if err != nil {
				panic(err)
			}
			interceptable := common.Construct(newGI.Interface(), "NewInterceptor")

			restIntercept, isRESTInterceptor := common.AsRESTInterceptor(interceptable)
			wsIntercept, isWSInterceptor := common.AsWSInterceptor(interceptable)
			if !isRESTInterceptor && !isWSInterceptor {
				panic(common.InterceptorShapeError(interceptable))
			}

			if isRESTInterceptor {
				for _, h := range app.module.RESTMainHandlers {
					httpMethod := routing.OperationsMapHTTPMethods[h.Method]
					endpoint := routing.MethodRouteVersionToPattern(httpMethod, h.Route, h.Version)
					mw := common.BuildHTTPInterceptMiddleware(endpoint, restIntercept)
					app.http.route.For([]string{httpMethod}, h.Route, h.Version)(mw)
				}
			}

			if isWSInterceptor {
				for _, h := range app.module.WSMainHandlers {
					mw := common.BuildWSInterceptMiddleware(h.EventName, wsIntercept)
					app.ws.eventMatcher.AddMiddlewares(h.EventName, mw)
				}
			}
		}
	}

	for _, mi := range app.module.RESTInterceptors {
		httpMethod := routing.OperationsMapHTTPMethods[mi.Method]
		endpoint := routing.MethodRouteVersionToPattern(httpMethod, mi.Route, mi.Version)
		mw := common.BuildHTTPInterceptMiddleware(endpoint, mi.Handler.(common.RESTIntercept))
		app.http.route.For([]string{httpMethod}, mi.Route, mi.Version)(mw)
	}

	for _, mi := range app.module.WSInterceptors {
		mw := common.BuildWSInterceptMiddleware(mi.EventName, mi.Handler.(common.WSIntercept))
		app.ws.eventMatcher.AddMiddlewares(mi.EventName, mw)
	}
}

func (app *App) initMainHandlers() {
	for _, h := range app.module.RESTMainHandlers {
		app.http.addMainHandler(h)
	}

	for _, h := range app.module.WSMainHandlers {
		app.ws.eventMatcher.AddInjectableHandler(h.EventName, h.Handler)
	}
}

func (app *App) releaseCtx(c *ctx.HTTPContext) {
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
	app.isDevtoolEnabled = true

	return app
}

func (app *App) EnableWS(cfg *WSConfig, middlewares ...common.MiddlewareFn) *App {
	app.isWSEnabled = true
	cfg.globalMiddlewares = middlewares
	app.wsConfig = cfg

	return app
}

func (app *App) UseLogger(logger common.Logger) *App {
	app.Logger = logger
	globalInterfaceByKey[injectableInterfaces[0]] = app.Logger

	return app
}

func (app *App) Get(p Provider) any {
	return app.injectedProviders[genProviderKey(p)]
}

func (app *App) Listen(port int) error {

	// REST logs
	var routeArr []string
	for _, h := range app.module.RESTMainHandlers {
		routeArr = append(routeArr, h.Pattern)
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

	if app.isWSEnabled {
		// WS logs
		for _, eventName := range app.module.WSMainHandlers {
			app.Logger.Info(
				"WebSocketEvent",
				"event", eventName.EventName,
			)
		}
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
