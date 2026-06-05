package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

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
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set(ctx.RequestID, "test-id-123")
	c.Request = r

	got := getContextID(c)
	if got != "test-id-123" {
		t.Error(test.DiffMessage(got, "test-id-123", "getContextID should use X-Request-Id header"))
	}
}

func TestGetContextIDGeneratesUUID(t *testing.T) {
	c1 := ctx.NewContext()
	c1.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c2 := ctx.NewContext()
	c2.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	id1 := getContextID(c1)
	id2 := getContextID(c2)

	if id1 == "" {
		t.Error(test.DiffMessage(id1, "non-empty UUID", "should generate UUID when header absent"))
	}
	if id1 == id2 {
		t.Error(test.DiffMessage(id1, "different UUID", "each call should produce a unique ID"))
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
