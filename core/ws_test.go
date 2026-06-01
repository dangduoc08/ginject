package core

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/middlewares/cors"
	"github.com/dangduoc08/ginject/testutils"
)

// mockCORSMW implements common.MiddlewareFn and corsOriginChecker without
// NewMiddleware(), so common.Construct leaves it uncached and each instance
// retains its own Allowed-origin set. All fields must be exported so
// injectDependencies can inspect them without panicking.
type mockCORSMW struct {
	Allowed map[string]bool
}

func (m mockCORSMW) Use(_ *ctx.Context, next ctx.Next) { next() }
func (m mockCORSMW) AllowedOrigin(origin string) bool  { return m.Allowed[origin] }

func TestNewWS_FieldsInitialized(t *testing.T) {
	ws := newWS()
	if ws.catchFnsMap == nil {
		t.Error(testutils.DiffMessage(nil, "map", "catchFnsMap should be initialized"))
	}
	if ws.eventMap == nil {
		t.Error(testutils.DiffMessage(nil, "map", "eventMap should be initialized"))
	}
	if ws.mainHandlerMap == nil {
		t.Error(testutils.DiffMessage(nil, "map", "mainHandlerMap should be initialized"))
	}
	if ws.eventToID == nil {
		t.Error(testutils.DiffMessage(nil, "map", "eventToID should be initialized"))
	}
}

func TestNewWS_CORSAllowOriginNilByDefault(t *testing.T) {
	ws := newWS()
	if ws.corsAllowOrigin != nil {
		t.Error(testutils.DiffMessage(ws.corsAllowOrigin, nil, "corsAllowOrigin should be nil before Create"))
	}
}

func TestCreate_WithoutCORS_CORSAllowOriginNil(t *testing.T) {
	app := New()
	app.Create(ModuleBuilder().Build())
	if app.ws.corsAllowOrigin != nil {
		t.Error(testutils.DiffMessage(app.ws.corsAllowOrigin, nil, "corsAllowOrigin should stay nil when no CORS middleware is bound"))
	}
}

func TestCreate_WithCORS_CORSAllowOriginSet(t *testing.T) {
	app := New()
	app.BindGlobalMiddlewares(cors.CORS{})
	app.Create(ModuleBuilder().Build())
	if app.ws.corsAllowOrigin == nil {
		t.Error(testutils.DiffMessage(nil, "func", "corsAllowOrigin should be wired when CORS middleware is bound"))
	}
}

func TestCreate_WithMockCORS_SpecificOriginAllowed(t *testing.T) {
	app := New()
	app.BindGlobalMiddlewares(mockCORSMW{Allowed: map[string]bool{"https://trusted.com": true}})
	app.Create(ModuleBuilder().Build())

	if !app.ws.corsAllowOrigin("https://trusted.com") {
		t.Error(testutils.DiffMessage(false, true, "listed origin should be Allowed"))
	}
}

func TestCreate_WithMockCORS_UnlistedOriginBlocked(t *testing.T) {
	app := New()
	app.BindGlobalMiddlewares(mockCORSMW{Allowed: map[string]bool{"https://trusted.com": true}})
	app.Create(ModuleBuilder().Build())

	if app.ws.corsAllowOrigin("https://evil.com") {
		t.Error(testutils.DiffMessage(true, false, "unlisted origin should be blocked"))
	}
}

func TestCreate_WithMockCORS_EmptyAllowedBlocksAll(t *testing.T) {
	app := New()
	app.BindGlobalMiddlewares(mockCORSMW{Allowed: map[string]bool{}})
	app.Create(ModuleBuilder().Build())

	if app.ws.corsAllowOrigin("https://example.com") {
		t.Error(testutils.DiffMessage(true, false, "empty Allowed map should block all origins"))
	}
}

func TestCreate_OnlyFirstCORSMiddlewareWired(t *testing.T) {
	app := New()
	app.BindGlobalMiddlewares(
		mockCORSMW{Allowed: map[string]bool{"https://first.com": true}},
		mockCORSMW{Allowed: map[string]bool{"https://second.com": true}},
	)
	app.Create(ModuleBuilder().Build())

	if !app.ws.corsAllowOrigin("https://first.com") {
		t.Error(testutils.DiffMessage(false, true, "first CORS middleware should be used"))
	}
	if app.ws.corsAllowOrigin("https://second.com") {
		t.Error(testutils.DiffMessage(true, false, "second CORS middleware should not be used"))
	}
}
