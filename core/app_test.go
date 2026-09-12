package core

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/log"
	"github.com/dangduoc08/ginject/memorybroker"
	"github.com/dangduoc08/ginject/trace"
	"github.com/dangduoc08/ginject/versioning"
	"golang.org/x/net/websocket"
)

type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any)  {}
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}
func (m *mockLogger) Fatal(msg string, args ...any) {}

func TestNew(t *testing.T) {
	app := New()
	if app == nil {
		t.Fatal(test.DiffMessage(nil, "*App", "New should not return nil"))
		return
	}
	if app.http.route == nil {
		t.Error(test.DiffMessage(nil, "router", "route not initialized"))
	}
	if app.http.catchFnsByRoute == nil {
		t.Error(test.DiffMessage(nil, "map", "catchHTTPFnsMap not initialized"))
	}
	if app.Logger != nil {
		t.Error(test.DiffMessage(app.Logger, nil, "Logger should be nil before Create"))
	}
}

func TestNewHasGlobalExceptionFilter(t *testing.T) {
	app := New()
	if len(app.globalExceptionFilters) == 0 {
		t.Error(test.DiffMessage(0, ">0", "New should register default global exception filter"))
	}
}

func TestGetContextIDIgnoresRequestIDHeader(t *testing.T) {
	c := ctx.NewHTTPContext()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(ctx.RequestID, "test-id-123")
	c.Init(httptest.NewRecorder(), r)

	if c.GetID() == "test-id-123" {
		t.Error(test.DiffMessage(c.GetID(), "<generated UUID, not the header value>", "Init must always generate its own ID and ignore the X-Request-Id header"))
	}
	if c.GetID() == "" {
		t.Error(test.DiffMessage(c.GetID(), "<non-empty UUID>", "Init must still generate an ID when a header is present"))
	}
}

func TestGetContextIDGeneratesUUID(t *testing.T) {
	c1 := ctx.NewHTTPContext()
	c1.Init(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	c2 := ctx.NewHTTPContext()
	c2.Init(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if c1.GetID() == "" {
		t.Error(test.DiffMessage(c1.GetID(), "non-empty UUID", "should generate UUID when header absent"))
	}
	if c1.GetID() == c2.GetID() {
		t.Error(test.DiffMessage(c1.GetID(), "different UUID", "each call should produce a unique ID"))
	}
}

func TestBindGlobalMiddlewaresChaining(t *testing.T) {
	app := New()
	result := app.BindGlobalMiddlewares()
	if result != app {
		t.Error(test.DiffMessage(result, app, "BindGlobalMiddlewares should return *App"))
	}
}

func TestBindGlobalGuardsChaining(t *testing.T) {
	app := New()
	result := app.BindGlobalGuards()
	if result != app {
		t.Error(test.DiffMessage(result, app, "BindGlobalGuards should return *App"))
	}
}

func TestBindGlobalInterceptorsChaining(t *testing.T) {
	app := New()
	result := app.BindGlobalInterceptors()
	if result != app {
		t.Error(test.DiffMessage(result, app, "BindGlobalInterceptors should return *App"))
	}
}

func TestBindGlobalExceptionFiltersChaining(t *testing.T) {
	app := New()
	result := app.BindGlobalExceptionFilters()
	if result != app {
		t.Error(test.DiffMessage(result, app, "BindGlobalExceptionFilters should return *App"))
	}
}

func TestEnableDevtoolChaining(t *testing.T) {
	app := New()
	result := app.EnableDevtool()
	if result != app {
		t.Error(test.DiffMessage(result, app, "EnableDevtool should return *App"))
	}
	if !app.isDevtoolEnabled {
		t.Error(test.DiffMessage(app.isDevtoolEnabled, true, "isDevtoolEnabled should be true after EnableDevtool"))
	}
}

func TestServeHTTPNotFound(t *testing.T) {
	app := New()
	app.Create(ModuleBuilder().Build())

	r := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Error(test.DiffMessage(w.Code, http.StatusNotFound, "unmatched route should return 404"))
	}
}

func TestServeHTTPSetsRequestIDHeader(t *testing.T) {
	app := New()
	app.Create(ModuleBuilder().Build())

	r := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	if w.Header().Get(ctx.RequestID) == "" {
		t.Error(test.DiffMessage("", "non-empty", "ServeHTTP should set X-Request-Id response header"))
	}
}

func TestServeHTTPSetsGeneratedRequestID(t *testing.T) {
	app := New()
	app.Create(ModuleBuilder().Build())

	const clientID = "client-supplied-request-id"
	r := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	r.Header.Set(ctx.RequestID, clientID)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	got := w.Header().Get(ctx.RequestID)
	if got == "" {
		t.Error(test.DiffMessage(got, "<non-empty UUID>", "ServeHTTP should set a generated request ID on the response"))
	}
	if got == clientID {
		t.Error(test.DiffMessage(got, "<generated UUID, not the client-supplied one>", "ServeHTTP should not echo back a client-supplied X-Request-Id"))
	}
}

type bodyLimitController struct{ common.HTTP }

func (c bodyLimitController) NewController() Controller { return c }

func (c bodyLimitController) CREATE(body ctx.Body) any {
	return ctx.Map{"got": body["k"]}
}

func newBodyLimitApp(t *testing.T, limit int64) *App {
	t.Helper()
	resetModuleGlobals()

	app := New()
	app.SetMaxRequestBodySize(limit)
	app.Create(ModuleBuilder().Controllers(bodyLimitController{}).Build())

	return app
}

func TestNew_AppliesDefaultBodyLimit(t *testing.T) {
	resetModuleGlobals()

	app := New()
	if app.maxRequestBodyBytes != DefaultMaxRequestBodyBytes {
		t.Error(test.DiffMessage(app.maxRequestBodyBytes, DefaultMaxRequestBodyBytes, "a new app must cap request bodies by default, not read them unbounded"))
	}
}

func TestServeHTTPBodyOverLimit_Returns413(t *testing.T) {
	app := newBodyLimitApp(t, 64)

	payload := `{"k":"` + strings.Repeat("a", 4096) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Error(test.DiffMessage(w.Code, http.StatusRequestEntityTooLarge, "a body past the configured cap must be rejected with 413, not buffered"))
	}
}

func TestServeHTTPBodyWithinLimit_Succeeds(t *testing.T) {
	app := newBodyLimitApp(t, 1<<20)

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"k":"ok"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatal(test.DiffMessage(w.Code, http.StatusCreated, "a body within the cap must be served normally"))
	}
	if !strings.Contains(w.Body.String(), `"ok"`) {
		t.Error(test.DiffMessage(w.Body.String(), `"ok"`, "the handler must still receive the parsed body"))
	}
}

func TestSetMaxRequestBodySize_ZeroDisablesCap(t *testing.T) {
	app := newBodyLimitApp(t, 0)

	payload := `{"k":"` + strings.Repeat("a", 4096) + `"}`
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payload))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	app.ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Error(test.DiffMessage(w.Code, http.StatusCreated, "an explicitly disabled cap must not reject large bodies"))
	}
}

func TestGetAfterCreate(t *testing.T) {
	app := New()
	app.Create(ModuleBuilder().Build())

	got := app.Get(&mockProvider{})
	if got != nil {
		t.Error(test.DiffMessage(got, nil, "Get for unregistered provider should return nil"))
	}
}

func TestUseLogger_SetsLogger(t *testing.T) {
	app := New()
	logger := &mockLogger{}
	result := app.UseLogger(logger)
	if app.Logger != logger {
		t.Error(test.DiffMessage(app.Logger, logger, "UseLogger should set Logger"))
	}
	if result != app {
		t.Error(test.DiffMessage(result, app, "UseLogger should return *App"))
	}
}

func TestServeHTTPConcurrent_NoDataRace(t *testing.T) {
	app := New()
	app.Create(ModuleBuilder().Build())

	const goroutines = 32
	const requestsPerGoroutine = 50

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				r := httptest.NewRequest(http.MethodGet, "/notfound", nil)
				w := httptest.NewRecorder()
				app.ServeHTTP(w, r)

				if w.Code != http.StatusNotFound {
					t.Error(test.DiffMessage(w.Code, http.StatusNotFound, "unmatched route should return 404 under concurrent load"))
				}
			}
		}()
	}
	wg.Wait()
}

type raceGlobalProvider struct{ Tag string }

func (p raceGlobalProvider) NewProvider() Provider { return p }

type raceGlobalMiddleware struct{ P raceGlobalProvider }

func (mw raceGlobalMiddleware) Use(_ *http.Request, _ http.ResponseWriter, next ctx.Next) { next() }

// TestConcurrentAppCreate_NoDataRace guards against a real, verified race:
// App.initLogger/UseLogger write to the package-level globalInterfaceByKey
// map with no lock, and injectDependencies (reached from every global
// middleware/guard/interceptor/exceptionFilter binding, from every module
// provider, and from every per-request pipeable-parameter resolution) reads
// globalProviderByKey/globalInterfaceByKey. Apps built and Created
// concurrently — a realistic scenario for parallel tests or multi-tenant
// setups — must not race on that shared state. Deliberately no controller
// here: this is about the provider-injection paths, not route registration
// (which has its own, separate global-state reset story via
// resetModuleGlobals/common.InsertedRoutes).
func TestConcurrentAppCreate_NoDataRace(t *testing.T) {
	resetModuleGlobals()

	const n = 8
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			app := New()
			app.BindGlobalMiddlewares(raceGlobalMiddleware{})
			app.Create(ModuleBuilder().
				Providers(raceGlobalProvider{}).
				Build())
		}()
	}
	wg.Wait()
}

func TestEnableVersioning_Chaining(t *testing.T) {
	app := New()
	result := app.EnableVersioning(versioning.Versioning{})
	if result != app {
		t.Error(test.DiffMessage(result, app, "EnableVersioning should return *App"))
	}
	if !app.http.isVersioningEnabled {
		t.Error(test.DiffMessage(app.http.isVersioningEnabled, true, "EnableVersioning should set isVersioningEnabled"))
	}
}

type panicMiddleware struct{}

func (panicMiddleware) Use(_ *http.Request, _ http.ResponseWriter, _ ctx.Next) {
	panic(exception.ForbiddenException("nope"))
}

type panicMiddlewareController struct {
	common.HTTP
}

func (c panicMiddlewareController) NewController() Controller { return c }
func (c panicMiddlewareController) READ_panicmiddleware() string {
	return "ok"
}

func TestGlobalMiddlewarePanic_CaughtByExceptionFilter(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.BindGlobalMiddlewares(panicMiddleware{})
	app.Create(ModuleBuilder().Controllers(panicMiddlewareController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/panicmiddleware", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Error(test.DiffMessage(w.Code, http.StatusForbidden, "a middleware panic must still be caught by the exception filter"))
	}
}

type denyGlobalGuard struct{}

func (denyGlobalGuard) CanActivate(_ *ctx.HTTPContext) bool { return false }

type shapelessGlobalGuard struct{}

type globalGuardController struct {
	common.HTTP
}

func (c globalGuardController) NewController() Controller { return c }
func (c globalGuardController) READ_globalguard() string  { return "ok" }

func TestGlobalGuard_DeniesHTTPRequest(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.BindGlobalGuards(denyGlobalGuard{})
	app.Create(ModuleBuilder().Controllers(globalGuardController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/globalguard", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Error(test.DiffMessage(w.Code, http.StatusForbidden, "a denying global guard must block the HTTP request"))
	}
}

func TestGlobalGuard_ShapelessGuardPanicsOnCreate(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.BindGlobalGuards(shapelessGlobalGuard{})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "a global guard with no CanActivate method must panic at Create"))
		}
	}()
	app.Create(ModuleBuilder().Controllers(globalGuardController{}).Build())
}

type traceMiddleware struct{}

func (traceMiddleware) Use(_ *http.Request, _ http.ResponseWriter, next ctx.Next) { next() }

type traceGuard struct{}

func (traceGuard) CanActivate(_ *ctx.HTTPContext) bool { return true }

type traceInterceptor struct{}

func (traceInterceptor) Intercept(_ *ctx.HTTPContext, agg *aggregation.Aggregation) any {
	return agg.Pipe()
}

type tracePipelineController struct {
	common.HTTP
	common.Middleware
	common.Guard
	common.Interceptor
}

func (c tracePipelineController) NewController() Controller {
	c.BindMiddleware(traceMiddleware{})
	c.BindGuard(traceGuard{})
	c.BindInterceptor(traceInterceptor{})
	return c
}
func (c tracePipelineController) READ_tracepipeline() string { return "ok" }

func TestTrace_EmitsPerStageAndComplete(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var stages []string
	var complete *trace.Event

	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageComplete {
			d := te
			complete = &d
			return
		}
		stages = append(stages, te.Stage)
	})

	app.Create(ModuleBuilder().Controllers(tracePipelineController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/tracepipeline", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()

	wantStages := map[string]bool{
		trace.StageMiddleware:     false,
		trace.StageGuard:          false,
		trace.StagePreInterceptor: false,
		trace.StageHandler:        false,
	}
	for _, s := range stages {
		wantStages[s] = true
	}
	for s, seen := range wantStages {
		if !seen {
			t.Errorf("expected a %q trace event to fire, stages seen: %v", s, stages)
		}
	}
	if complete == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a complete trace event to fire"))
	}
	if complete.Code != http.StatusOK {
		t.Error(test.DiffMessage(complete.Code, http.StatusOK, "complete trace event should carry the final status code"))
	}
}

const traceSlowPipeSleep = 30 * time.Millisecond

type traceSlowPipeDTO struct{}

func (d traceSlowPipeDTO) Transform(ctx.Query, common.ArgumentMetadata) any {
	time.Sleep(traceSlowPipeSleep)
	return traceSlowPipeDTO{}
}

type tracePipeController struct {
	common.HTTP
}

func (c tracePipeController) NewController() Controller              { return c }
func (c tracePipeController) READ_tracepipe(traceSlowPipeDTO) string { return "ok" }

func TestTrace_PipeStageExcludedFromHandlerDuration(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var pipeEvent, handlerEvent *trace.Event
	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StagePipe:
			d := te
			pipeEvent = &d
		case trace.StageHandler:
			d := te
			handlerEvent = &d
		}
	})

	app.Create(ModuleBuilder().Controllers(tracePipeController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/tracepipe", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()

	if pipeEvent == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a pipe trace event to fire"))
	}
	if pipeEvent.Duration < traceSlowPipeSleep {
		t.Error(test.DiffMessage(pipeEvent.Duration, ">= "+traceSlowPipeSleep.String(), "pipe trace event should report the pipe's own execution time"))
	}
	if handlerEvent == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a handler trace event to fire"))
	}
	if handlerEvent.Duration >= traceSlowPipeSleep {
		t.Error(test.DiffMessage(handlerEvent.Duration, "< "+traceSlowPipeSleep.String(), "handler trace duration should exclude the pipe's own execution time"))
	}
}

type tracePanicController struct {
	common.HTTP
}

func (c tracePanicController) NewController() Controller { return c }
func (c tracePanicController) READ_tracepanic() string {
	panic(exception.InternalServerErrorException("boom"))
}

func TestTrace_ExceptionFilterStageFiresOnPanic(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var sawExceptionFilter bool
	var complete *trace.Event

	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		if te.Stage == trace.StageExceptionFilter {
			sawExceptionFilter = true
		}
		if te.Stage == trace.StageComplete {
			d := te
			complete = &d
		}
	})

	app.Create(ModuleBuilder().Controllers(tracePanicController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/tracepanic", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()

	if !sawExceptionFilter {
		t.Error(test.DiffMessage(false, true, "expected an exceptionFilter trace event to fire for a panicking handler"))
	}
	if complete == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a complete trace event to fire even on error"))
	}
	if complete.Code != http.StatusInternalServerError {
		t.Error(test.DiffMessage(complete.Code, http.StatusInternalServerError, "complete trace event should carry the error status code"))
	}
}

type tracePanickingPipeDTO struct{}

func (d tracePanickingPipeDTO) Transform(ctx.Query, common.ArgumentMetadata) any {
	panic(exception.BadRequestException("bad query"))
}

type tracePipePanicController struct {
	common.HTTP
}

func (c tracePipePanicController) NewController() Controller                        { return c }
func (c tracePipePanicController) READ_tracepipepanic(tracePanickingPipeDTO) string { return "ok" }

func TestTrace_PipePanicStillEmitsPipeEventAndReachesExceptionFilter(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var pipeEvent *trace.Event
	var handlerEvent *trace.Event
	var sawExceptionFilter bool
	var complete *trace.Event

	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StagePipe:
			d := te
			pipeEvent = &d
		case trace.StageHandler:
			d := te
			handlerEvent = &d
		case trace.StageExceptionFilter:
			sawExceptionFilter = true
		case trace.StageComplete:
			d := te
			complete = &d
		}
	})

	app.Create(ModuleBuilder().Controllers(tracePipePanicController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/tracepipepanic", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()

	if pipeEvent == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a pipe trace event even though Transform panicked"))
	}
	if pipeEvent.Transport != trace.TransportHTTP {
		t.Error(test.DiffMessage(pipeEvent.Transport, trace.TransportHTTP, "pipe trace event transport"))
	}
	if handlerEvent != nil {
		t.Error(test.DiffMessage(handlerEvent, "<no handler event>", "no StageHandler event should fire when the panic happened before the handler body ever ran"))
	}
	if !sawExceptionFilter {
		t.Error(test.DiffMessage(false, true, "the pipe's panic should still reach the exception filter"))
	}
	if complete == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a complete trace event to fire even when a pipe panics"))
	}
	if complete.Code != http.StatusBadRequest {
		t.Error(test.DiffMessage(complete.Code, http.StatusBadRequest, "the pipe's exception should determine the final status code"))
	}
}

type traceHandlerPanicAfterPipeController struct {
	common.HTTP
}

func (c traceHandlerPanicAfterPipeController) NewController() Controller { return c }
func (c traceHandlerPanicAfterPipeController) READ_tracehandlerpanicafterpipe(traceSlowPipeDTO) string {
	panic(exception.InternalServerErrorException("handler exploded"))
}

func TestTrace_HandlerPanicAfterSuccessfulPipeStillExcludesPipeDuration(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var pipeEvent, handlerEvent *trace.Event
	var complete *trace.Event

	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StagePipe:
			d := te
			pipeEvent = &d
		case trace.StageHandler:
			d := te
			handlerEvent = &d
		case trace.StageComplete:
			d := te
			complete = &d
		}
	})

	app.Create(ModuleBuilder().Controllers(traceHandlerPanicAfterPipeController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/tracehandlerpanicafterpipe", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()

	if pipeEvent == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected the pipe trace event since the pipe itself succeeded"))
	}
	if pipeEvent.Duration < traceSlowPipeSleep {
		t.Error(test.DiffMessage(pipeEvent.Duration, ">= "+traceSlowPipeSleep.String(), "pipe trace event should report the pipe's own execution time"))
	}
	if handlerEvent == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a handler trace event since the handler body started executing before it panicked"))
	}
	if handlerEvent.Duration >= traceSlowPipeSleep {
		t.Error(test.DiffMessage(handlerEvent.Duration, "< "+traceSlowPipeSleep.String(), "handler trace duration should still exclude the pipe's execution time even when the handler body itself panics"))
	}
	if complete == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a complete trace event to fire"))
	}
	if complete.Code != http.StatusInternalServerError {
		t.Error(test.DiffMessage(complete.Code, http.StatusInternalServerError, "the handler's exception should determine the final status code"))
	}
}

type multiInterceptorA struct{}

func (multiInterceptorA) Intercept(_ *ctx.HTTPContext, agg *aggregation.Aggregation) any {
	return agg.Pipe()
}

type multiInterceptorB struct{}

func (multiInterceptorB) Intercept(_ *ctx.HTTPContext, agg *aggregation.Aggregation) any {
	return agg.Pipe()
}

type multiInterceptorC struct{}

func (multiInterceptorC) Intercept(_ *ctx.HTTPContext, agg *aggregation.Aggregation) any {
	return agg.Pipe()
}

type multiInterceptorController struct {
	common.HTTP
	common.Interceptor
}

func (c multiInterceptorController) NewController() Controller {
	c.BindInterceptor(multiInterceptorA{})
	c.BindInterceptor(multiInterceptorB{})
	c.BindInterceptor(multiInterceptorC{})
	return c
}
func (c multiInterceptorController) READ_multiintercept() string { return "ok" }

func TestTrace_InterceptorPrePostEventsMatchInCountAndOrder(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var pre []string
	var post []string

	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StagePreInterceptor:
			pre = append(pre, te.Name)
		case trace.StagePostInterceptor:
			post = append(post, te.Name)
		}
	})

	app.Create(ModuleBuilder().Controllers(multiInterceptorController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/multiintercept", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()

	if len(pre) != 3 {
		t.Fatalf("expected 3 pre-interceptor events, got %d: %v", len(pre), pre)
	}
	if len(post) != len(pre) {
		t.Fatalf("expected %d post-interceptor events to match the pre-interceptor count, got %d: %v", len(pre), len(post), post)
	}

	wantPost := make([]string, len(pre))
	for i, name := range pre {
		wantPost[len(pre)-1-i] = name
	}
	if post[0] != wantPost[0] || post[1] != wantPost[1] || post[2] != wantPost[2] {
		t.Error(test.DiffMessage(post, wantPost, "post-interceptor events must fire in reverse execution order of the pre-interceptor events"))
	}
}

type shortCircuitInterceptor struct{}

func (shortCircuitInterceptor) Intercept(_ *ctx.HTTPContext, agg *aggregation.Aggregation) any {
	return "short-circuited"
}

type shortCircuitController struct {
	common.HTTP
	common.Interceptor
}

func (c shortCircuitController) NewController() Controller {
	c.BindInterceptor(multiInterceptorA{})
	c.BindInterceptor(shortCircuitInterceptor{})
	c.BindInterceptor(multiInterceptorC{})
	return c
}
func (c shortCircuitController) READ_shortcircuit() string { return "ok" }

func TestTrace_InterceptorPrePostEventsMatchWhenShortCircuited(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var preCount, postCount int

	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StagePreInterceptor:
			preCount++
		case trace.StagePostInterceptor:
			postCount++
		}
	})

	app.Create(ModuleBuilder().Controllers(shortCircuitController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/shortcircuit", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	mu.Lock()
	defer mu.Unlock()

	if preCount != 3 {
		t.Fatalf("expected 3 pre-interceptor events, got %d", preCount)
	}
	if postCount != preCount {
		t.Error(test.DiffMessage(postCount, preCount, "post-interceptor event count must match pre-interceptor event count even when an interceptor short-circuits"))
	}
}

func TestTrace_InterceptorNoListener_NoPostEventOverhead(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.Create(ModuleBuilder().Controllers(multiInterceptorController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/multiintercept", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	if w.Body.String() != "ok" {
		t.Error(test.DiffMessage(w.Body.String(), "ok", "request should complete normally with no trace listener attached"))
	}
}

func TestTrace_WSHandshakeEmitsStageCompleteWithHTTPTransport(t *testing.T) {
	resetModuleGlobals()
	app := New()

	var mu sync.Mutex
	var complete *trace.Event
	var preMiddlewareTransport string
	app.event.On(trace.EventName, func(args ...any) {
		mu.Lock()
		defer mu.Unlock()
		te := args[0].(trace.Event)
		switch te.Stage {
		case trace.StageMiddleware:
			preMiddlewareTransport = te.Transport
		case trace.StageComplete:
			d := te
			complete = &d
		}
	})

	app.EnableWS(&WSConfig{Path: "/ws"}, traceMiddleware{})
	app.Create(ModuleBuilder().Build())

	server := httptest.NewServer(app)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
	conn, err := websocket.Dial(wsURL, "", server.URL)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	mu.Lock()
	defer mu.Unlock()

	if preMiddlewareTransport != trace.TransportHTTP {
		t.Error(test.DiffMessage(preMiddlewareTransport, trace.TransportHTTP, "handshake middleware trace event must report HTTP transport"))
	}
	if complete == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected a StageComplete trace event for the WS handshake"))
	}
	if complete.Transport != trace.TransportHTTP {
		t.Error(test.DiffMessage(complete.Transport, trace.TransportHTTP, "WS handshake StageComplete must report HTTP transport, not WS, since the connection has not upgraded yet"))
	}
}

type capturingMockLogger struct {
	mu   sync.Mutex
	args []any
}

func (m *capturingMockLogger) capture(args []any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.args = args
}

func (m *capturingMockLogger) lastArgs() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.args
}

func (m *capturingMockLogger) Debug(_ string, args ...any) { m.capture(args) }
func (m *capturingMockLogger) Info(_ string, args ...any)  { m.capture(args) }
func (m *capturingMockLogger) Warn(_ string, args ...any)  { m.capture(args) }
func (m *capturingMockLogger) Error(_ string, args ...any) { m.capture(args) }
func (m *capturingMockLogger) Fatal(_ string, args ...any) { m.capture(args) }

func TestUseLogOptions_SetsLogOptions(t *testing.T) {
	app := New()
	opts := &log.LogOptions{MaskFields: []string{"password"}}
	result := app.UseLogOptions(opts)
	if app.LogOptions != opts {
		t.Error(test.DiffMessage(app.LogOptions, opts, "UseLogOptions should set LogOptions"))
	}
	if result != app {
		t.Error(test.DiffMessage(result, app, "UseLogOptions should return *App"))
	}
}

func TestInitLogger_CustomLoggerAutomaticallyGetsTagBehavior(t *testing.T) {
	type secret struct {
		Value string `log:"value"`
		skip  string
	}

	capturing := &capturingMockLogger{}
	app := New()
	app.UseLogger(capturing)
	app.Create(ModuleBuilder().Build())

	app.Logger.Info("test", "s", secret{Value: "visible", skip: "hidden"})

	got, ok := capturing.lastArgs()[1].(map[string]any)
	if !ok {
		t.Fatalf("expected the struct to be expanded into a map[string]any, got %T", capturing.lastArgs()[1])
	}
	if got["value"] != "visible" {
		t.Error(test.DiffMessage(got["value"], "visible", "tagged field should be logged under its tag name"))
	}
	if _, exists := got["skip"]; exists {
		t.Error(test.DiffMessage(true, false, "untagged unexported field must not be logged"))
	}
}

func TestInitLogger_CustomLoggerAutomaticallyGetsMaskBehavior(t *testing.T) {
	capturing := &capturingMockLogger{}
	app := New()
	app.UseLogger(capturing)
	app.UseLogOptions(&log.LogOptions{MaskFields: []string{"password"}})
	app.Create(ModuleBuilder().Build())

	app.Logger.Info("test", "password", "secret")

	if capturing.lastArgs()[1] != "[REDACTED]" {
		t.Error(test.DiffMessage(capturing.lastArgs()[1], "[REDACTED]", "masking rules from UseLogOptions should apply to a custom Logger automatically"))
	}
}

type capturedLogCall struct {
	msg  string
	args []any
}

type allCallsCapturingLogger struct {
	mu    sync.Mutex
	calls []capturedLogCall
}

func (l *allCallsCapturingLogger) capture(msg string, args []any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, capturedLogCall{msg: msg, args: args})
}

func (l *allCallsCapturingLogger) findCall(msg string) *capturedLogCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := range l.calls {
		if l.calls[i].msg == msg {
			return &l.calls[i]
		}
	}
	return nil
}

func (l *allCallsCapturingLogger) findAllCalls(msg string) []capturedLogCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	var found []capturedLogCall
	for _, c := range l.calls {
		if c.msg == msg {
			found = append(found, c)
		}
	}
	return found
}

func (l *allCallsCapturingLogger) Debug(msg string, args ...any) { l.capture(msg, args) }
func (l *allCallsCapturingLogger) Info(msg string, args ...any)  { l.capture(msg, args) }
func (l *allCallsCapturingLogger) Warn(msg string, args ...any)  { l.capture(msg, args) }
func (l *allCallsCapturingLogger) Error(msg string, args ...any) { l.capture(msg, args) }
func (l *allCallsCapturingLogger) Fatal(msg string, args ...any) { l.capture(msg, args) }

var testListenModule = func() *Module {
	return ModuleBuilder().Build()
}

func TestListen_LogsModuleNameWhenResolved(t *testing.T) {
	logger := &allCallsCapturingLogger{}
	app := New()
	app.UseLogger(logger)
	app.Create(testListenModule())

	if err := app.Listen(-1); err == nil {
		t.Fatal(test.DiffMessage(nil, "error", "Listen(-1) should fail immediately with an invalid port"))
	}

	call := logger.findCall("InstanceLoader")
	if call == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil", "expected an InstanceLoader log entry"))
	}
	want := []any{"module", "core.testListenModule initialized"}
	if len(call.args) != 2 || call.args[0] != want[0] || call.args[1] != want[1] {
		t.Error(test.DiffMessage(call.args, want, "module log args"))
	}
}

func TestListen_SkipsModuleLogWhenNameUnresolved(t *testing.T) {
	logger := &allCallsCapturingLogger{}
	app := New()
	app.UseLogger(logger)

	m := ModuleBuilder().Build()
	m.Name = ""
	app.Create(m)

	if err := app.Listen(-1); err == nil {
		t.Fatal(test.DiffMessage(nil, "error", "Listen(-1) should fail immediately with an invalid port"))
	}

	if call := logger.findCall("InstanceLoader"); call != nil {
		t.Error(test.DiffMessage(call.msg, "<no InstanceLoader log>", "module log should be skipped when Name is unresolved"))
	}
}

var testListenChildModule = func() *Module {
	return ModuleBuilder().Build()
}

var testListenParentModule = func() *Module {
	return ModuleBuilder().Imports(testListenChildModule).Build()
}

func TestListen_LogsNestedModuleNamesRecursively(t *testing.T) {
	logger := &allCallsCapturingLogger{}
	app := New()
	app.UseLogger(logger)
	app.Create(testListenParentModule())

	if err := app.Listen(-1); err == nil {
		t.Fatal(test.DiffMessage(nil, "error", "Listen(-1) should fail immediately with an invalid port"))
	}

	calls := logger.findAllCalls("InstanceLoader")
	gotNames := make(map[string]bool, len(calls))
	for _, c := range calls {
		gotNames[c.args[1].(string)] = true
	}

	wantNames := []string{
		"core.testListenParentModule initialized",
		"core.testListenChildModule initialized",
	}
	for _, name := range wantNames {
		if !gotNames[name] {
			t.Error(test.DiffMessage(gotNames, name, "expected nested module name to be logged"))
		}
	}
}

func TestInitDevtool_DoesNotSpawnAServerGoroutine(t *testing.T) {
	resetModuleGlobals()

	before := runtime.NumGoroutine()

	app := New()
	app.EnableDevtool()
	app.Create(ModuleBuilder().Build())

	if app.devtool == nil {
		t.Fatal("EnableDevtool must still build the devtool snapshot")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before+1 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Errorf("devtool must not leave a goroutine behind while its transport is unimplemented: %d before, %d after",
		before, runtime.NumGoroutine())
}

type appTraceSink struct {
	mu     sync.Mutex
	events []trace.Event
}

func (s *appTraceSink) attach(app *App) {
	app.event.On(trace.EventName, func(args ...any) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.events = append(s.events, args[0].(trace.Event))
	})
}

func (s *appTraceSink) all() []trace.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]trace.Event, len(s.events))
	copy(out, s.events)
	return out
}

func (s *appTraceSink) completes() []trace.Event {
	var out []trace.Event
	for _, e := range s.all() {
		if e.Stage == trace.StageComplete {
			out = append(out, e)
		}
	}
	return out
}

func (s *appTraceSink) waitForComplete(t testing.TB, operation string) trace.Event {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range s.completes() {
			if e.Operation == operation {
				return e
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("no StageComplete trace event for operation %q; saw %+v", operation, s.completes())
	return trace.Event{}
}

var handshakeOrder = struct {
	mu    sync.Mutex
	steps []string
}{}

func recordHandshakeStep(name string) {
	handshakeOrder.mu.Lock()
	defer handshakeOrder.mu.Unlock()
	handshakeOrder.steps = append(handshakeOrder.steps, name)
}

func handshakeSteps() []string {
	handshakeOrder.mu.Lock()
	defer handshakeOrder.mu.Unlock()
	return append([]string{}, handshakeOrder.steps...)
}

func resetHandshakeOrder() {
	handshakeOrder.mu.Lock()
	defer handshakeOrder.mu.Unlock()
	handshakeOrder.steps = nil
}

type firstHandshakeMiddleware struct{}

func (firstHandshakeMiddleware) Use(_ *http.Request, _ http.ResponseWriter, next ctx.Next) {
	recordHandshakeStep("first")
	next()
}

type denyingHandshakeMiddleware struct{}

func (denyingHandshakeMiddleware) Use(_ *http.Request, w http.ResponseWriter, _ ctx.Next) {
	recordHandshakeStep("second")
	w.WriteHeader(http.StatusUnauthorized)
}

type thirdHandshakeMiddleware struct{}

func (thirdHandshakeMiddleware) Use(_ *http.Request, _ http.ResponseWriter, next ctx.Next) {
	recordHandshakeStep("third")
	next()
}

type globalHandshakeMiddleware struct{}

func (globalHandshakeMiddleware) Use(r *http.Request, _ http.ResponseWriter, next ctx.Next) {
	handshakeGlobalRan.Store(true)
	next()
}

var handshakeGlobalRan syncBool

type syncBool struct {
	mu sync.Mutex
	v  bool
}

func (b *syncBool) Store(v bool) { b.mu.Lock(); b.v = v; b.mu.Unlock() }

func (b *syncBool) Load() bool { b.mu.Lock(); defer b.mu.Unlock(); return b.v }

func TestHandshake_RunsGlobalHTTPMiddlewaresBeforeUpgrade(t *testing.T) {
	resetModuleGlobals()
	handshakeGlobalRan.Store(false)

	app := New()
	app.BindGlobalMiddlewares(globalHandshakeMiddleware{})
	app.EnableWS(&WSConfig{Path: "/ws"})
	app.Create(ModuleBuilder().Build())

	server := httptest.NewServer(app)
	defer server.Close()

	conn, err := websocket.Dial(wsURLOf(server), "", server.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if !handshakeGlobalRan.Load() {
		t.Error(test.DiffMessage(false, true, "a WebSocket handshake is an HTTP request, so BindGlobalMiddlewares must run on it — otherwise the upgrade path silently skips global auth"))
	}
}

func TestHandshake_MiddlewareOrderIsPreservedAndRejectionStopsTheChain(t *testing.T) {
	resetModuleGlobals()
	resetHandshakeOrder()

	app := New()
	app.EnableWS(&WSConfig{Path: "/ws"},
		firstHandshakeMiddleware{},
		denyingHandshakeMiddleware{},
		thirdHandshakeMiddleware{},
	)
	app.Create(ModuleBuilder().Build())

	server := httptest.NewServer(app)
	defer server.Close()

	if _, err := websocket.Dial(wsURLOf(server), "", server.URL); err == nil {
		t.Fatal(test.DiffMessage(nil, "error", "a middleware that does not call next() must reject the handshake"))
	}

	order := handshakeSteps()
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Error(test.DiffMessage(order, []string{"first", "second"}, "middleware must run in declaration order and stop at the one that rejects"))
	}
}

func TestHandshake_RejectionIsLoggedWithAFailureStatusNotA101(t *testing.T) {
	resetModuleGlobals()
	resetHandshakeOrder()

	app := New()
	sink := &appTraceSink{}
	sink.attach(app)

	app.EnableWS(&WSConfig{Path: "/ws"}, denyingHandshakeMiddleware{})
	app.Create(ModuleBuilder().Build())

	server := httptest.NewServer(app)
	defer server.Close()

	_, _ = websocket.Dial(wsURLOf(server), "", server.URL)

	e := sink.waitForComplete(t, trace.OperationHandshake)
	if e.Status != trace.StatusRejected {
		t.Error(test.DiffMessage(e.Status, trace.StatusRejected, "a rejected handshake must be distinguishable from a successful one in the access log"))
	}
	if e.Code == http.StatusSwitchingProtocols {
		t.Error(test.DiffMessage(e.Code, "not 101", "a rejected handshake must not be logged as a successful 101 upgrade"))
	}
}

func TestHandshake_SuccessIsLoggedAsHandshakeWithA101(t *testing.T) {
	resetModuleGlobals()

	app := New()
	sink := &appTraceSink{}
	sink.attach(app)

	app.EnableWS(&WSConfig{Path: "/ws"})
	app.Create(ModuleBuilder().Build())

	server := httptest.NewServer(app)
	defer server.Close()

	conn, err := websocket.Dial(wsURLOf(server), "", server.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	e := sink.waitForComplete(t, trace.OperationHandshake)
	if e.Status != trace.StatusOK {
		t.Error(test.DiffMessage(e.Status, trace.StatusOK, "a successful handshake must be logged as ok"))
	}
	if e.Code != http.StatusSwitchingProtocols {
		t.Error(test.DiffMessage(e.Code, http.StatusSwitchingProtocols, "a successful handshake must be logged with the 101 it actually returned"))
	}
	if e.Transport != trace.TransportHTTP {
		t.Error(test.DiffMessage(e.Transport, trace.TransportHTTP, "the handshake is still an HTTP request"))
	}
}

func TestHandshake_MalformedUpgradeStillProducesAnAccessLogEntry(t *testing.T) {
	resetModuleGlobals()

	app := New()
	sink := &appTraceSink{}
	sink.attach(app)

	app.EnableWS(&WSConfig{Path: "/ws"})
	app.Create(ModuleBuilder().Build())

	server := httptest.NewServer(app)
	defer server.Close()

	resp, err := http.Get(server.URL + "/ws")
	if err == nil {
		_ = resp.Body.Close()
	}

	e := sink.waitForComplete(t, trace.OperationHandshake)
	if e.Status != trace.StatusFailed {
		t.Error(test.DiffMessage(e.Status, trace.StatusFailed, "a request that never completes the upgrade must still be logged, or failed handshakes are invisible in production"))
	}
}

func TestHandshake_OriginRejectionIsReportedAsRejected(t *testing.T) {
	resetModuleGlobals()

	app := New()
	sink := &appTraceSink{}
	sink.attach(app)

	app.EnableWS(&WSConfig{Path: "/ws", AllowedOrigins: []string{"https://app.example.com"}})
	app.Create(ModuleBuilder().Build())

	server := httptest.NewServer(app)
	defer server.Close()

	if _, err := websocket.Dial(wsURLOf(server), "", "https://evil.example"); err == nil {
		t.Fatal(test.DiffMessage(nil, "error", "an unlisted Origin must not be upgraded"))
	}

	e := sink.waitForComplete(t, trace.OperationHandshake)
	if e.Status != trace.StatusRejected {
		t.Error(test.DiffMessage(e.Status, trace.StatusRejected, "an origin rejection must be logged as a rejection"))
	}
}

type recordingBroker struct {
	inner memorybroker.Broker

	mu         sync.Mutex
	subscribed []string
	published  []string
}

func newRecordingBroker() *recordingBroker {
	return &recordingBroker{inner: memorybroker.NewMemoryBroker()}
}

func (b *recordingBroker) Subscribe(topic string, h memorybroker.MessageHandler) (memorybroker.Subscription, error) {
	b.mu.Lock()
	b.subscribed = append(b.subscribed, topic)
	b.mu.Unlock()
	return b.inner.Subscribe(topic, h)
}

func (b *recordingBroker) Unsubscribe(sub memorybroker.Subscription) error {
	return b.inner.Unsubscribe(sub)
}

func (b *recordingBroker) Publish(topic string, payload any) error {
	b.mu.Lock()
	b.published = append(b.published, topic)
	b.mu.Unlock()
	return b.inner.Publish(topic, payload)
}

func (b *recordingBroker) PublishAsync(topic string, payload any) error {
	return b.inner.PublishAsync(topic, payload)
}

func (b *recordingBroker) Close() error { return b.inner.Close() }

func (b *recordingBroker) snapshot() ([]string, []string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string{}, b.subscribed...), append([]string{}, b.published...)
}

func TestBroker_CanBeReplacedWithoutTouchingTheWebSocketLayer(t *testing.T) {
	resetModuleGlobals()

	custom := newRecordingBroker()

	app := New()
	app.UseBroker(custom)
	app.EnableWS(&WSConfig{Path: "/ws"})
	app.Create(ModuleBuilder().Build())

	if app.broker != memorybroker.Broker(custom) {
		t.Fatal(test.DiffMessage("memorybroker", "custom", "UseBroker must replace the app broker; without this seam the WebSocket layer can never scale past one process"))
	}
	if *app.ws.connmgr.Broker != memorybroker.Broker(custom) {
		t.Error(test.DiffMessage("memorybroker", "custom", "the WebSocket connection manager must fan out through the injected broker"))
	}
}

func wsURLOf(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
}

type auditWSController struct {
	common.WS
	common.Guard
}

func (c auditWSController) NewController() Controller {
	c.BindGuard(auditTenantGuard{})
	return c
}

func (c auditWSController) SUBSCRIBE_chat_ANY(topic ctx.WSTopic) ctx.Map {
	return ctx.Map{"echo": string(topic)}
}

type auditTenantGuard struct{}

func (auditTenantGuard) CanActivate(c *ctx.WSContext) bool {
	return c.Topic() != "chat.forbidden"
}

func dialAudit(t testing.TB, server *httptest.Server) *websocket.Conn {
	t.Helper()

	conn, err := websocket.Dial("ws"+server.URL[len("http"):]+"/ws", "", server.URL)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func expectFrame(t testing.TB, conn *websocket.Conn, want WSPayloadType) WSPayload {
	t.Helper()

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var p WSPayload
	if err := websocket.JSON.Receive(conn, &p); err != nil {
		t.Fatalf("receive (wanted %v): %v", want, err)
	}
	if p.Type != want {
		t.Fatalf("expected %v frame, got %v (%+v)", want, p.Type, p)
	}
	return p
}

func newAuditApp(t testing.TB) (*App, *httptest.Server, *appTraceSink) {
	t.Helper()
	resetModuleGlobals()

	app := New()
	sink := &appTraceSink{}
	sink.attach(app)

	app.EnableWS(&WSConfig{Path: "/ws", PingInterval: time.Hour})
	app.Create(ModuleBuilder().Controllers(auditWSController{}).Build())

	server := httptest.NewServer(app)
	t.Cleanup(server.Close)

	return app, server, sink
}

// TestIntegration_FullClientJourney drives the real HTTP server the way a
// browser client would: handshake, connected frame, subscribe, publish, read
// both the handler response and the broker fan-out, then unsubscribe.
func TestIntegration_FullClientJourney(t *testing.T) {
	app, server, sink := newAuditApp(t)

	conn := dialAudit(t, server)
	defer func() { _ = conn.Close() }()

	connected := expectFrame(t, conn, TypeConnected)
	if connected.ID == "" {
		t.Fatal(test.DiffMessage("", "a connection id", "the server must hand the client its connection id"))
	}

	if err := websocket.JSON.Send(conn, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.room1"}}); err != nil {
		t.Fatal(err)
	}
	ack := expectFrame(t, conn, TypeAck)
	if ack.ID != "s1" {
		t.Error(test.DiffMessage(ack.ID, "s1", "the ack must echo the request id"))
	}

	if err := websocket.JSON.Send(conn, WSPayload{ID: "p1", Type: TypePublish, Topic: []string{"chat.room1"}, Message: map[string]any{"text": "hi"}}); err != nil {
		t.Fatal(err)
	}

	response := expectFrame(t, conn, TypeResponse)
	if m, ok := response.Message.(map[string]any); !ok || m["echo"] != "chat.room1" {
		t.Error(test.DiffMessage(response.Message, map[string]any{"echo": "chat.room1"}, "the handler must receive the concrete topic and its answer must reach the publisher"))
	}

	fanOut := expectFrame(t, conn, TypeEvent)
	if fanOut.Pattern != "chat.room1" {
		t.Error(test.DiffMessage(fanOut.Pattern, "chat.room1", "the fan-out event must name the subscription that delivered it"))
	}

	expectFrame(t, conn, TypeAck)

	if err := websocket.JSON.Send(conn, WSPayload{ID: "u1", Type: TypeUnsubscribe, Topic: []string{"chat.room1"}}); err != nil {
		t.Fatal(err)
	}
	if unack := expectFrame(t, conn, TypeAck); unack.ID != "u1" {
		t.Error(test.DiffMessage(unack.ID, "u1", "unsubscribe must be acked"))
	}

	waitFor(t, func() bool { return app.WSStats().Subscriptions == 0 })

	var sawSubscribe, sawUnsubscribe, sawPublish, sawHandshake bool
	for _, e := range sink.completes() {
		switch e.Operation {
		case trace.OperationHandshake:
			sawHandshake = true
		case trace.OperationSubscribe:
			sawSubscribe = true
		case trace.OperationUnsubscribe:
			sawUnsubscribe = true
		case trace.OperationPublish:
			sawPublish = true
		}
	}
	if !sawHandshake || !sawSubscribe || !sawUnsubscribe || !sawPublish {
		t.Error(test.DiffMessage(
			[]bool{sawHandshake, sawSubscribe, sawUnsubscribe, sawPublish},
			[]bool{true, true, true, true},
			"all four phases (handshake, subscribe, unsubscribe, publish) must produce access-log entries",
		))
	}
}

func TestIntegration_GuardDeniesPerTopicOverTheWire(t *testing.T) {
	_, server, _ := newAuditApp(t)

	conn := dialAudit(t, server)
	defer func() { _ = conn.Close() }()
	expectFrame(t, conn, TypeConnected)

	if err := websocket.JSON.Send(conn, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.forbidden"}}); err != nil {
		t.Fatal(err)
	}
	expectFrame(t, conn, TypeError)

	if err := websocket.JSON.Send(conn, WSPayload{ID: "s2", Type: TypeSubscribe, Topic: []string{"chat.allowed"}}); err != nil {
		t.Fatal(err)
	}
	expectFrame(t, conn, TypeAck)
}

func TestIntegration_PublishWithoutSubscribeIsRejected(t *testing.T) {
	_, server, _ := newAuditApp(t)

	conn := dialAudit(t, server)
	defer func() { _ = conn.Close() }()
	expectFrame(t, conn, TypeConnected)

	if err := websocket.JSON.Send(conn, WSPayload{ID: "p1", Type: TypePublish, Topic: []string{"chat.room1"}, Message: "x"}); err != nil {
		t.Fatal(err)
	}
	got := expectFrame(t, conn, TypeError)

	results, ok := got.Message.([]any)
	if !ok || len(results) != 1 {
		t.Fatalf("expected one per-topic result, got %+v", got.Message)
	}
	if m := results[0].(map[string]any); m["code"] != float64(4001) {
		t.Error(test.DiffMessage(m["code"], 4001, "publishing to a topic the connection never subscribed to must be rejected as not-subscribed"))
	}
}

func TestIntegration_DisconnectReleasesConnectionAndSubscriptions(t *testing.T) {
	app, server, _ := newAuditApp(t)

	conn := dialAudit(t, server)
	expectFrame(t, conn, TypeConnected)

	if err := websocket.JSON.Send(conn, WSPayload{ID: "s1", Type: TypeSubscribe, Topic: []string{"chat.a", "chat.b"}}); err != nil {
		t.Fatal(err)
	}
	expectFrame(t, conn, TypeAck)

	waitFor(t, func() bool {
		s := app.WSStats()
		return s.Connections == 1 && s.Subscriptions == 2
	})

	_ = conn.Close()

	waitFor(t, func() bool {
		s := app.WSStats()
		return s.Connections == 0 && s.Subscriptions == 0
	})
}

func TestIntegration_TwoClientsSeeEachOthersMessages(t *testing.T) {
	_, server, _ := newAuditApp(t)

	a := dialAudit(t, server)
	defer func() { _ = a.Close() }()
	b := dialAudit(t, server)
	defer func() { _ = b.Close() }()

	expectFrame(t, a, TypeConnected)
	expectFrame(t, b, TypeConnected)

	for _, c := range []*websocket.Conn{a, b} {
		if err := websocket.JSON.Send(c, WSPayload{ID: "s", Type: TypeSubscribe, Topic: []string{"chat.shared"}}); err != nil {
			t.Fatal(err)
		}
		expectFrame(t, c, TypeAck)
	}

	if err := websocket.JSON.Send(a, WSPayload{ID: "p", Type: TypePublish, Topic: []string{"chat.shared"}, Message: "from-a"}); err != nil {
		t.Fatal(err)
	}

	event := expectFrame(t, b, TypeEvent)
	if event.Message != "from-a" {
		t.Error(test.DiffMessage(event.Message, "from-a", "the other subscriber must receive the published message"))
	}
	if len(event.Topic) != 1 || event.Topic[0] != "chat.shared" {
		t.Error(test.DiffMessage(event.Topic, []string{"chat.shared"}, "the event must name the concrete topic"))
	}
}

func waitFor(t testing.TB, cond func() bool) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met within 3s")
}
