package core

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/dangduoc08/ginject/accesslog"
	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/devtool"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/routing"
	"github.com/dangduoc08/ginject/trace"
	"github.com/dangduoc08/ginject/versioning"
	"golang.org/x/net/websocket"
)

type App struct {
	http      *HTTP
	ctxPool   sync.Pool
	wsCtxPool sync.Pool

	event *event.Event

	isDevtoolEnabled bool
	devtool          *devtool.Devtool

	isAccessLogEnabled bool
	accesslog          *accesslog.AccessLog

	ws          *WS
	wsConfig    *WSConfig
	isWSEnabled bool

	module *Module

	globalMiddlewares      []common.MiddlewareFn
	globalGuarders         []common.Guarder
	globalInterceptors     []common.Interceptable
	globalExceptionFilters []common.ExceptionFilterable
	injectedProviders      map[string]Provider

	broker broker.Broker

	Logger     common.Logger
	LogOptions *log.LogOptions
}

const (
	httpContextKey  = "/*ctx.HTTPContext"
	wsContextKey    = "/*ctx.WSContext"
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
	publisherKey    = "github.com/dangduoc08/ginject/common/common.Publisher"
)

// knownHTTPDependencyKeys is the set of dependency-type keys the framework
// can resolve for a HTTP handler parameter (see getHTTPDependency); values
// are unused and always 1.
var knownHTTPDependencyKeys = map[string]int{
	httpContextKey:            1,
	requestKey:                1,
	responseKey:               1,
	bodyKey:                   1,
	formKey:                   1,
	queryKey:                  1,
	headerKey:                 1,
	paramKey:                  1,
	fileKey:                   1,
	nextKey:                   1,
	redirectKey:               1,
	publisherKey:              1,
	common.ContextPipeableKey: 1,
	common.BodyPipeableKey:    1,
	common.FormPipeableKey:    1,
	common.QueryPipeableKey:   1,
	common.HeaderPipeableKey:  1,
	common.ParamPipeableKey:   1,
	common.FilePipeableKey:    1,
}

// knownWSDependencyKeys is the set of dependency-type keys the framework
// can resolve for a WS handler parameter (see getWSDependency); WS handlers
// only get WS-relevant dependencies, not the HTTP-only ones above.
var knownWSDependencyKeys = map[string]int{
	wsContextKey:                1,
	wsConnectionKey:             1,
	wsPayloadKey:                1,
	nextKey:                     1,
	publisherKey:                1,
	common.WSPayloadPipeableKey: 1,
}

type WithValueKey = common.WithValueKey

func New() *App {
	app := &App{
		http:   newHTTP(),
		ws:     nil,
		event:  event.NewEvent(),
		broker: broker.NewBroker(),
		ctxPool: sync.Pool{
			New: func() any {
				return ctx.NewHTTPContext()
			},
		},
		wsCtxPool: sync.Pool{
			New: func() any {
				return ctx.NewWSContext()
			},
		},
	}

	globalInterfaceByKey.Store(publisherKey, common.Publisher(
		newPublisher(app.broker),
	))

	app.BindGlobalExceptionFilters(globalHTTPExceptionFilter{}, globalWSExceptionFilter{})

	return app
}

func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := app.ctxPool.Get().(*ctx.HTTPContext)
	c.Init(w, r)

	if app.isWSEnabled && app.ws.isWSPath(r.URL.Path) {
		app.ws.upgrade(w, r, websocket.Server{
			Handshake: func(wsCfg *websocket.Config, r *http.Request) error {
				defer app.releaseCtx(c)

				c.Request = r
				c.Status(http.StatusSwitchingProtocols)
				err := app.ws.handshake(c)
				if app.event.HasListeners(trace.EventName) {
					app.event.Emit(trace.EventName, trace.Event{
						ID:        c.GetID(),
						Stage:     trace.StageComplete,
						Transport: trace.TransportHTTP,
						Operation: c.Method,
						Target:    c.URL.Path,
						Code:      c.Code,
						Duration:  time.Since(c.Timestamp),
					})
				}
				return err
			},
			Handler: websocket.Handler(app.ws.handleRequest),
		})

		return
	}

	defer app.releaseCtx(c)
	c.ResponseWriter.Header().Set(ctx.RequestID, c.GetID())
	app.http.handleRequest(c)
	if app.event.HasListeners(trace.EventName) {
		app.event.Emit(trace.EventName, trace.Event{
			ID:        c.GetID(),
			Stage:     trace.StageComplete,
			Transport: trace.TransportHTTP,
			Operation: c.Method,
			Target:    c.URL.Path,
			Code:      c.Code,
			Duration:  time.Since(c.Timestamp),
		})
	}
}

// Order matters here and is easy to break by accident:
//
//  1. initProviders must run first — it builds injectedProviders and
//     app.module (parsed from m), which everything below reads from.
//  2. initWS must run right after that, and BEFORE initExceptionFilters/
//     initGuards/initInterceptors/initMainHandlers — those functions write
//     into app.ws.catchFnsByEvent / app.ws.eventMatcher whenever app.module
//     has WS handlers. app.ws stays nil if EnableWS was never called, so
//     every one of those call sites is guarded with app.ws != nil rather
//     than assuming it's set.
//  3. initDevtool must run last — it reads the fully-populated
//     module/route state to build the devtool snapshot.
//
// Do not reorder these calls without re-checking every init* function for
// an unguarded app.ws.* access.
func (app *App) Create(m *Module) {
	app.initLogger()
	injectedProviders := app.initProviders(m)
	app.initWS(injectedProviders)
	app.initMiddlewares(injectedProviders)
	app.initExceptionFilters(injectedProviders)
	app.initGuards(injectedProviders)
	app.initInterceptors(injectedProviders)
	app.initMainHandlers()
	app.initDevtool()
	app.initAccessLog()
}

func (app *App) initWS(injectedProviders map[string]Provider) {
	if !app.isWSEnabled {
		return
	}

	app.wsConfig.injectedProviders = injectedProviders
	app.wsConfig.logger = app.Logger
	app.wsConfig.event = app.event
	app.wsConfig.broker = &app.broker
	app.wsConfig.resolveAndCallHandler = func(f any, c *ctx.WSContext) []reflect.Value {
		return invokeWSHandlerByProviders(f, injectedProviders, c)
	}
	app.wsConfig.newCtx = func() *ctx.WSContext { return app.wsCtxPool.Get().(*ctx.WSContext) }
	app.wsConfig.releaseCtx = app.releaseWSCtx
	app.ws = NewWS(app.wsConfig)
	app.ws.emitComplete = func(c *ctx.WSContext, operation, target string) {
		if !app.event.HasListeners(trace.EventName) {
			return
		}
		app.event.Emit(trace.EventName, trace.Event{
			ID:        c.GetID(),
			Stage:     trace.StageComplete,
			Transport: trace.TransportWS,
			Operation: operation,
			Target:    target,
			Duration:  time.Since(c.Timestamp),
		})
	}
}

func (app *App) initAccessLog() {
	if !app.isAccessLogEnabled {
		return
	}

	app.accesslog = accesslog.NewAccessLog(&accesslog.AccessLogConfig{
		Event:  app.event,
		Logger: app.Logger,
	})
}

func (app *App) initLogger() {
	if app.Logger == nil {
		app.Logger = log.NewLog(app.LogOptions)
	} else {
		var maskFields []string
		if app.LogOptions != nil {
			maskFields = app.LogOptions.MaskFields
		}
		app.Logger = log.WrapLogger(app.Logger, maskFields)
	}
	globalInterfaceByKey.Store(injectableInterfaces[0], app.Logger)
}

func (app *App) initProviders(m *Module) map[string]Provider {
	app.module = m.NewModule()

	injectedProviders := make(map[string]Provider, len(app.module.providers))
	for _, provider := range app.module.providers {
		injectedProviders[genProviderKey(provider)] = provider
	}
	app.injectedProviders = injectedProviders

	nameByPattern := make(map[string]string, len(app.module.HTTPMainHandlers))
	for _, mh := range app.module.HTTPMainHandlers {
		nameByPattern[mh.Pattern] = mh.Name
	}

	resolveAndCallHandler := func(pattern string, f any, c *ctx.HTTPContext) []reflect.Value {
		if !app.event.HasListeners(trace.EventName) {
			return invokeHTTPHandlerByProviders(f, injectedProviders, c)
		}

		start := time.Now()
		defer func() {
			app.event.Emit(trace.EventName, trace.Event{
				ID:       c.GetID(),
				Stage:    trace.StageHandler,
				Name:     nameByPattern[pattern],
				Duration: time.Since(start),
			})
		}()
		return invokeHTTPHandlerByProviders(f, injectedProviders, c)
	}
	app.http.resolveAndCallHandler = resolveAndCallHandler

	return injectedProviders
}

func (app *App) initExceptionFilters(injectedProviders map[string]Provider) {
	for i := len(app.module.HTTPExceptionFilters) - 1; i >= 0; i-- {
		ef := app.module.HTTPExceptionFilters[i]
		httpMethod := routing.OperationsMapHTTPMethods[ef.Method]
		endpoint := routing.MethodRouteVersionToPattern(httpMethod, ef.Route, ef.Version)
		catchFn := traceHTTPCatch(app.event, ef.Name, ef.Handler.(common.HTTPCatch))
		app.http.catchFnsByRoute[endpoint] = append(app.http.catchFnsByRoute[endpoint], catchFn)
	}

	if app.ws != nil {
		for i := len(app.module.WSExceptionFilters) - 1; i >= 0; i-- {
			ef := app.module.WSExceptionFilters[i]
			catchFn := traceWSCatch(app.event, ef.Name, ef.Handler.(common.WSCatch))
			app.ws.catchFnsByEvent[ef.EventName] = append(app.ws.catchFnsByEvent[ef.EventName], catchFn)
		}
	}

	if len(app.globalExceptionFilters) > 0 {
		for i := len(app.globalExceptionFilters) - 1; i >= 0; i-- {
			gef := app.globalExceptionFilters[i]
			name := reflect.TypeOf(gef).String()
			newGef, err := injectDependencies(gef, "exceptionFilter", injectedProviders)
			if err != nil {
				panic(err)
			}
			exceptionFilterable := common.Construct(newGef.Interface(), "NewExceptionFilter")

			httpCatch, isHTTPFilter := common.AsHTTPExceptionFilter(exceptionFilterable)
			wsCatch, isWSFilter := common.AsWSExceptionFilter(exceptionFilterable)
			if !isHTTPFilter && !isWSFilter {
				panic(common.ExceptionFilterShapeError(exceptionFilterable))
			}

			if isHTTPFilter {
				tracedHTTPCatch := traceHTTPCatch(app.event, name, httpCatch)
				for _, h := range app.module.HTTPMainHandlers {
					httpMethod := routing.OperationsMapHTTPMethods[h.Method]
					endpoint := routing.MethodRouteVersionToPattern(httpMethod, h.Route, h.Version)
					app.http.catchFnsByRoute[endpoint] = append(app.http.catchFnsByRoute[endpoint], tracedHTTPCatch)
				}
			}

			if isWSFilter && app.ws != nil {
				tracedWSCatch := traceWSCatch(app.event, name, wsCatch)
				for _, h := range app.module.WSMainHandlers {
					app.ws.catchFnsByEvent[h.EventName] = append(app.ws.catchFnsByEvent[h.EventName], tracedWSCatch)
				}
			}
		}
	}
}

func (app *App) initMiddlewares(injectedProviders map[string]Provider) {
	if len(app.globalMiddlewares) > 0 {
		for _, gm := range app.globalMiddlewares {
			name := reflect.TypeOf(gm).String()
			newGM, err := injectDependencies(gm, "middleware", injectedProviders)
			if err != nil {
				panic(err)
			}
			gm = common.Construct(newGM.Interface(), "NewMiddleware").(common.MiddlewareFn)
			mw := buildUseMiddleware(gm.Use, app.event, name, trace.TransportHTTP)
			app.http.route.Use(mw)
		}
	}

	for _, rm := range app.module.HTTPMiddlewares {
		mw := buildUseMiddleware(rm.Handler.(common.Use), app.event, rm.Name, trace.TransportHTTP)
		httpMethod := routing.OperationsMapHTTPMethods[rm.Method]
		app.http.route.For([]string{httpMethod}, rm.Route, rm.Version)(mw)
	}
}

func (app *App) initGuards(injectedProviders map[string]Provider) {
	if len(app.globalGuarders) > 0 {
		for _, gg := range app.globalGuarders {
			name := reflect.TypeOf(gg).String()
			newGG, err := injectDependencies(gg, "guard", injectedProviders)
			if err != nil {
				panic(err)
			}
			guarder := common.Construct(newGG.Interface(), "NewGuard")

			httpCanActivate, isHTTPGuard := common.AsHTTPGuard(guarder)
			wsCanActivate, isWSGuard := common.AsWSGuard(guarder)
			if !isHTTPGuard && !isWSGuard {
				panic(common.GuardShapeError(guarder))
			}

			if isHTTPGuard {
				mw := traceHTTPHandler(app.event, trace.StageGuard, name, common.BuildHTTPGuardMiddleware(httpCanActivate))
				for _, h := range app.module.HTTPMainHandlers {
					httpMethod := routing.OperationsMapHTTPMethods[h.Method]
					app.http.route.For([]string{httpMethod}, h.Route, h.Version)(mw)
				}
			}

			if isWSGuard && app.ws != nil {
				mw := traceWSHandler(app.event, trace.StageGuard, name, common.BuildWSGuardMiddleware(wsCanActivate))
				for _, h := range app.module.WSMainHandlers {
					app.ws.eventMatcher.AddMiddlewares(h.EventName, mw)
				}
			}
		}
	}

	for _, mg := range app.module.HTTPGuards {
		mw := traceHTTPHandler(app.event, trace.StageGuard, mg.Name, common.BuildHTTPGuardMiddleware(mg.Handler.(common.HTTPCanActivate)))
		httpMethod := routing.OperationsMapHTTPMethods[mg.Method]
		app.http.route.For([]string{httpMethod}, mg.Route, mg.Version)(mw)
	}

	if app.ws != nil {
		for _, mg := range app.module.WSGuards {
			mw := traceWSHandler(app.event, trace.StageGuard, mg.Name, common.BuildWSGuardMiddleware(mg.Handler.(common.WSCanActivate)))
			app.ws.eventMatcher.AddMiddlewares(mg.EventName, mw)
		}
	}
}

func tagInterceptorName(ev *event.Event, endpoint, name string, h ctx.HTTPHandler) ctx.HTTPHandler {
	return func(c *ctx.HTTPContext) {
		h(c)
		if !ev.HasListeners(trace.EventName) {
			return
		}
		if aggregations, ok := c.Context().Value(WithValueKey(endpoint)).([]*aggregation.Aggregation); ok && len(aggregations) > 0 {
			aggregations[len(aggregations)-1].Name = name
		}
	}
}

func tagWSInterceptorName(ev *event.Event, eventName, name string, h ctx.WSHandler) ctx.WSHandler {
	return func(c *ctx.WSContext) {
		h(c)
		if !ev.HasListeners(trace.EventName) {
			return
		}
		if aggregations, ok := c.Context().Value(WithValueKey(eventName)).([]*aggregation.Aggregation); ok && len(aggregations) > 0 {
			aggregations[len(aggregations)-1].Name = name
		}
	}
}

func (app *App) initInterceptors(injectedProviders map[string]Provider) {
	app.http.emitPostInterceptor = func(c *ctx.HTTPContext, name string, duration time.Duration) {
		if !app.event.HasListeners(trace.EventName) {
			return
		}
		app.event.Emit(trace.EventName, trace.Event{
			ID:       c.GetID(),
			Stage:    trace.StagePostInterceptor,
			Name:     name,
			Duration: duration,
		})
	}

	if app.ws != nil {
		app.ws.emitPostInterceptor = func(c *ctx.WSContext, name string, duration time.Duration) {
			if !app.event.HasListeners(trace.EventName) {
				return
			}
			app.event.Emit(trace.EventName, trace.Event{
				ID:        c.GetID(),
				Stage:     trace.StagePostInterceptor,
				Name:      name,
				Transport: trace.TransportWS,
				Duration:  duration,
			})
		}
	}

	if len(app.globalInterceptors) > 0 {
		for _, gi := range app.globalInterceptors {
			name := reflect.TypeOf(gi).String()
			newGI, err := injectDependencies(gi, "interceptor", injectedProviders)
			if err != nil {
				panic(err)
			}
			interceptable := common.Construct(newGI.Interface(), "NewInterceptor")

			httpIntercept, isHTTPInterceptor := common.AsHTTPInterceptor(interceptable)
			wsIntercept, isWSInterceptor := common.AsWSInterceptor(interceptable)
			if !isHTTPInterceptor && !isWSInterceptor {
				panic(common.InterceptorShapeError(interceptable))
			}

			if isHTTPInterceptor {
				for _, h := range app.module.HTTPMainHandlers {
					httpMethod := routing.OperationsMapHTTPMethods[h.Method]
					endpoint := routing.MethodRouteVersionToPattern(httpMethod, h.Route, h.Version)
					mw := traceHTTPHandler(app.event, trace.StagePreInterceptor, name, common.BuildHTTPInterceptMiddleware(endpoint, httpIntercept))
					mw = tagInterceptorName(app.event, endpoint, name, mw)
					app.http.route.For([]string{httpMethod}, h.Route, h.Version)(mw)
				}
			}

			if isWSInterceptor && app.ws != nil {
				for _, h := range app.module.WSMainHandlers {
					mw := traceWSHandler(app.event, trace.StagePreInterceptor, name, common.BuildWSInterceptMiddleware(h.EventName, wsIntercept))
					mw = tagWSInterceptorName(app.event, h.EventName, name, mw)
					app.ws.eventMatcher.AddMiddlewares(h.EventName, mw)
				}
			}
		}
	}

	for _, mi := range app.module.HTTPInterceptors {
		httpMethod := routing.OperationsMapHTTPMethods[mi.Method]
		endpoint := routing.MethodRouteVersionToPattern(httpMethod, mi.Route, mi.Version)
		mw := traceHTTPHandler(app.event, trace.StagePreInterceptor, mi.Name, common.BuildHTTPInterceptMiddleware(endpoint, mi.Handler.(common.HTTPIntercept)))
		mw = tagInterceptorName(app.event, endpoint, mi.Name, mw)
		app.http.route.For([]string{httpMethod}, mi.Route, mi.Version)(mw)
	}

	if app.ws != nil {
		for _, mi := range app.module.WSInterceptors {
			mw := traceWSHandler(app.event, trace.StagePreInterceptor, mi.Name, common.BuildWSInterceptMiddleware(mi.EventName, mi.Handler.(common.WSIntercept)))
			mw = tagWSInterceptorName(app.event, mi.EventName, mi.Name, mw)
			app.ws.eventMatcher.AddMiddlewares(mi.EventName, mw)
		}
	}
}

func (app *App) initMainHandlers() {
	for _, h := range app.module.HTTPMainHandlers {
		app.http.addMainHandler(h)
	}

	if app.ws != nil {
		for _, h := range app.module.WSMainHandlers {
			app.ws.eventMatcher.AddInjectableHandler(h.EventName, h.Handler)
		}
	}
}

func (app *App) releaseCtx(c *ctx.HTTPContext) {
	c.Reset()
	app.ctxPool.Put(c)
}

func (app *App) releaseWSCtx(c *ctx.WSContext) {
	c.Reset()
	app.wsCtxPool.Put(c)
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

func (app *App) EnableAccessLog() *App {
	app.isAccessLogEnabled = true

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
	globalInterfaceByKey.Store(injectableInterfaces[0], app.Logger)

	return app
}

func (app *App) UseLogOptions(opts *log.LogOptions) *App {
	app.LogOptions = opts

	return app
}

func (app *App) Get(p Provider) any {
	return app.injectedProviders[genProviderKey(p)]
}

func (app *App) Listen(port int) error {

	// HTTP logs
	var routeArr []string
	for _, h := range app.module.HTTPMainHandlers {
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

func (app *App) initDevtool() {
	if !app.isDevtoolEnabled {
		return
	}

	devtoolBuilder := devtool.DevtoolBuilder()

	app.devtool = devtoolBuilder.
		AddExceptionFilters(app.globalExceptionFilters, app.module.HTTPExceptionFilters).
		AddMiddlewares(app.globalMiddlewares, app.module.HTTPMiddlewares).
		AddGuarders(app.globalGuarders, app.module.HTTPGuards).
		AddInterceptors(app.globalInterceptors, app.module.HTTPInterceptors).
		AddVersioning(app.http.versioning).
		AddHTTPMainHandlers(app.module.HTTPMainHandlers).
		Build()

	go app.devtool.Serve()
}
