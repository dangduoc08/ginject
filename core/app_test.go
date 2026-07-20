package core

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/versioning"
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
		t.Error(test.DiffMessage(nil, "map", "catchRESTFnsMap not initialized"))
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

func TestGetContextIDFromHeader(t *testing.T) {
	c := ctx.NewHTTPContext()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(ctx.RequestID, "test-id-123")
	c.Init(httptest.NewRecorder(), r)

	if c.GetID() != "test-id-123" {
		t.Error(test.DiffMessage(c.GetID(), "test-id-123", "Init should use X-Request-Id header"))
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

func TestServeHTTPPropagatesRequestID(t *testing.T) {
	app := New()
	app.Create(ModuleBuilder().Build())

	const fixedID = "fixed-request-id"
	r := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	r.Header.Set(ctx.RequestID, fixedID)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	if w.Header().Get(ctx.RequestID) != fixedID {
		t.Error(test.DiffMessage(w.Header().Get(ctx.RequestID), fixedID, "ServeHTTP should echo provided X-Request-Id"))
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
	common.REST
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
	common.REST
}

func (c globalGuardController) NewController() Controller { return c }
func (c globalGuardController) READ_globalguard() string  { return "ok" }

func TestGlobalGuard_DeniesRESTRequest(t *testing.T) {
	resetModuleGlobals()
	app := New()
	app.BindGlobalGuards(denyGlobalGuard{})
	app.Create(ModuleBuilder().Controllers(globalGuardController{}).Build())

	r := httptest.NewRequest(http.MethodGet, "/globalguard", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Error(test.DiffMessage(w.Code, http.StatusForbidden, "a denying global guard must block the REST request"))
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
