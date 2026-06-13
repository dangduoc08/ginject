package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/ctx"
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
	if app.http.catchFnsMap == nil {
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
	c := ctx.NewContext()
	c.Broker = broker.New()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(ctx.RequestID, "test-id-123")
	c.Init(httptest.NewRecorder(), r)

	if c.GetID() != "test-id-123" {
		t.Error(test.DiffMessage(c.GetID(), "test-id-123", "Init should use X-Request-Id header"))
	}
}

func TestGetContextIDGeneratesUUID(t *testing.T) {
	c1 := ctx.NewContext()
	c1.Broker = broker.New()
	c1.Init(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	c2 := ctx.NewContext()
	c2.Broker = broker.New()
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
	if !app.isEnableDevtool {
		t.Error(test.DiffMessage(app.isEnableDevtool, true, "isEnableDevtool should be true after EnableDevtool"))
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

func TestEnableVersioning_Chaining(t *testing.T) {
	app := New()
	result := app.EnableVersioning(versioning.Versioning{})
	if result != app {
		t.Error(test.DiffMessage(result, app, "EnableVersioning should return *App"))
	}
	if !app.http.isEnableVersioning {
		t.Error(test.DiffMessage(app.http.isEnableVersioning, true, "EnableVersioning should set isEnableVersioning"))
	}
}
