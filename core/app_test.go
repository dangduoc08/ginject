package core

import (
	"net/http"
	"net/http/httptest"
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

func (c tracePipeController) NewController() Controller { return c }
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
