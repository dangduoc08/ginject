package core

import (
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
)

// NewModule leans on package-level state (mainModulePtr, globalProviderByKey,
// providerSingletonByKey, staticModuleByDynamicPtr, globalPrefixesByController) that persists
// across the whole test binary, plus common.InsertedRoutes/InsertedEvents.
// Every test below must start from a clean slate or it will observe leftover
// state from whichever test happened to run first in the package.
func resetModuleGlobals() {
	mainModulePtr = 0
	modulesInjectedFromMain = nil
	staticModuleByDynamicPtr = make(map[uintptr]*Module)
	globalPrefixesByController = make(map[string][]string)
	globalProviderByKey = sync.Map{}
	providerSingletonByKey = make(map[string]Provider)
	common.InsertedRoutes = make(map[string]string)
	common.InsertedEvents = make(map[string]string)
}

// ---------------------------------------------------------------------------
// provider bootstrap: hoisting, singleton construction, global promotion,
// dynamic module resolution
// ---------------------------------------------------------------------------

var mtConstructOrder []string

type mtOrderedProviderA struct{}

func (p mtOrderedProviderA) NewProvider() Provider {
	mtConstructOrder = append(mtConstructOrder, "A")
	return p
}

type mtOrderedProviderB struct{ A mtOrderedProviderA }

func (p mtOrderedProviderB) NewProvider() Provider {
	mtConstructOrder = append(mtConstructOrder, "B")
	return p
}

func TestNewModule_ProviderHoisting_DependencyConstructedBeforeDependent(t *testing.T) {
	resetModuleGlobals()
	mtConstructOrder = nil

	// declared out of dependency order on purpose: B depends on A but is
	// listed first.
	m := ModuleBuilder().
		Providers(mtOrderedProviderB{}, mtOrderedProviderA{}).
		Build()

	m.NewModule()

	if len(mtConstructOrder) != 2 || mtConstructOrder[0] != "A" || mtConstructOrder[1] != "B" {
		t.Error(test.DiffMessage(mtConstructOrder, []string{"A", "B"}, "a provider's dependency must construct before the dependent, regardless of declaration order"))
	}
}

var mtOnceConstructCount int

type mtOnceProvider struct{}

func (p mtOnceProvider) NewProvider() Provider {
	mtOnceConstructCount++
	return p
}

func TestNewModule_ProviderConstructedOnce_AcrossDuplicateStaticImports(t *testing.T) {
	resetModuleGlobals()
	mtOnceConstructCount = 0

	child := ModuleBuilder().Providers(mtOnceProvider{}).Build()
	parentA := ModuleBuilder().Imports(child).Build()
	parentB := ModuleBuilder().Imports(child).Build()
	root := ModuleBuilder().Imports(parentA, parentB).Build()

	root.NewModule()

	if mtOnceConstructCount != 1 {
		t.Error(test.DiffMessage(mtOnceConstructCount, 1, "a provider reachable via two static-module paths must construct exactly once"))
	}
}

type mtGlobalSubProvider struct{}

func (p mtGlobalSubProvider) NewProvider() Provider { return p }

func TestNewModule_StaticSubmoduleGlobalProvider_PromotedToGlobal(t *testing.T) {
	resetModuleGlobals()

	child := ModuleBuilder().Providers(mtGlobalSubProvider{}).Build()
	child.IsGlobal = true
	root := ModuleBuilder().Imports(child).Build()

	root.NewModule()

	key := genFieldKey(reflect.TypeOf(mtGlobalSubProvider{}))
	if _, ok := globalProviderByKey.Load(key); !ok {
		t.Error(test.DiffMessage(nil, "non-nil provider", "a provider from an IsGlobal static submodule imported by the main module must be promoted to globalProviderByKey"))
	}
}

type mtDynMissingProvider struct{}

func (p mtDynMissingProvider) NewProvider() Provider { return p }

func mtDynamicModuleNeedingProvider(_ mtDynMissingProvider) *Module {
	return ModuleBuilder().Build()
}

func TestNewModule_DynamicModule_MissingGlobalDependencyPanics(t *testing.T) {
	resetModuleGlobals()
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "a dynamic module factory whose argument has no global provider must panic during bootstrap"))
		}
	}()

	root := ModuleBuilder().Imports(mtDynamicModuleNeedingProvider).Build()
	root.NewModule()
}

type mtDedupController struct{}

func (c mtDedupController) NewController() Controller { return c }

func TestNewModule_StaticModuleControllerDedup_AcrossSharedImport(t *testing.T) {
	resetModuleGlobals()

	shared := mtDedupController{}
	child := ModuleBuilder().Controllers(shared).Build()
	parentA := ModuleBuilder().Imports(child).Build()
	parentB := ModuleBuilder().Imports(child).Build()
	root := ModuleBuilder().Imports(parentA, parentB).Build()

	root.NewModule()

	if len(root.controllers) != 1 {
		t.Error(test.DiffMessage(len(root.controllers), 1, "a controller reachable via two static-module paths must be deduped"))
	}
}

// ---------------------------------------------------------------------------
// shared provider/guard fixtures for the REST/WS layer-injection tests
// ---------------------------------------------------------------------------

type mtLocalProvider struct{ Source string }

func (p mtLocalProvider) NewProvider() Provider { p.Source = "local"; return p }

type mtGlobalOnlyProvider struct{ Source string }

func (p mtGlobalOnlyProvider) NewProvider() Provider { return p }

type mtInterfaceOnlyProvider struct{ Source string }

func (p mtInterfaceOnlyProvider) NewProvider() Provider { return p }

type mtPassthroughValue struct{ Tag string }

type mtUnresolvedProvider struct{}

func (p mtUnresolvedProvider) NewProvider() Provider { return p }

type mtPriorityGuard struct {
	Local       mtLocalProvider
	GlobalOnly  mtGlobalOnlyProvider
	IfaceOnly   mtInterfaceOnlyProvider
	Passthrough mtPassthroughValue
}

var mtPriorityGuardSeen mtPriorityGuard

func (g mtPriorityGuard) CanActivate(_ *ctx.HTTPContext) bool {
	mtPriorityGuardSeen = g
	return true
}

type mtUnexportedFieldGuard struct {
	hidden mtLocalProvider //nolint:unused // exists only so reflection finds an unexported field
}

func (g mtUnexportedFieldGuard) CanActivate(_ *ctx.HTTPContext) bool { return true }

type mtUnresolvedFieldGuard struct{ Missing mtUnresolvedProvider }

func (g mtUnresolvedFieldGuard) CanActivate(_ *ctx.HTTPContext) bool { return true }

type mtSimpleMiddleware struct{ P mtLocalProvider }

var mtSimpleMiddlewareSeen mtSimpleMiddleware

func (mw mtSimpleMiddleware) Use(_ *http.Request, _ http.ResponseWriter, next ctx.Next) {
	mtSimpleMiddlewareSeen = mw
	next()
}

type mtSimpleInterceptor struct{ P mtLocalProvider }

var mtSimpleInterceptorSeen mtSimpleInterceptor

func (ic mtSimpleInterceptor) Intercept(_ *ctx.HTTPContext, _ *aggregation.Aggregation) any {
	mtSimpleInterceptorSeen = ic
	return nil
}

type mtExFilterOneField struct{ P mtLocalProvider }

var mtExFilterOneFieldSeen mtExFilterOneField

func (e mtExFilterOneField) Catch(_ *ctx.HTTPContext, _ *exception.Exception) {
	mtExFilterOneFieldSeen = e
}

func seedPriorityChainGlobals() {
	// a stale/other-module global registration for the SAME type the module
	// also declares locally - local must still win.
	globalProviderByKey.Store(genFieldKey(reflect.TypeOf(mtLocalProvider{})), mtLocalProvider{Source: "stale-global"})
	globalProviderByKey.Store(genFieldKey(reflect.TypeOf(mtGlobalOnlyProvider{})), mtGlobalOnlyProvider{Source: "global"})
	globalInterfaceByKey.Store(genFieldKey(reflect.TypeOf(mtInterfaceOnlyProvider{})), mtInterfaceOnlyProvider{Source: "interface"})
}

func assertPriorityChainSeen(t *testing.T, label string) {
	t.Helper()
	if mtPriorityGuardSeen.Local.Source != "local" {
		t.Error(test.DiffMessage(mtPriorityGuardSeen.Local.Source, "local", label+": local module provider must win over a global provider of the same type"))
	}
	if mtPriorityGuardSeen.GlobalOnly.Source != "global" {
		t.Error(test.DiffMessage(mtPriorityGuardSeen.GlobalOnly.Source, "global", label+": field with no local provider must fall back to globalProviderByKey"))
	}
	if mtPriorityGuardSeen.IfaceOnly.Source != "interface" {
		t.Error(test.DiffMessage(mtPriorityGuardSeen.IfaceOnly.Source, "interface", label+": field with no local/global provider must fall back to globalInterfaceByKey"))
	}
	if mtPriorityGuardSeen.Passthrough.Tag != "original" {
		t.Error(test.DiffMessage(mtPriorityGuardSeen.Passthrough.Tag, "original", label+": non-Provider field must pass through the bound instance's original value"))
	}
}

// ---------------------------------------------------------------------------
// REST controller processing
// ---------------------------------------------------------------------------

type mtPrefixedController struct{ common.REST }

func (c mtPrefixedController) NewController() Controller   { return c }
func (c mtPrefixedController) READ_mtprefixtarget() string { return "ok" }

func TestNewModule_RESTMainHandler_RegisteredWithModulePrefix(t *testing.T) {
	resetModuleGlobals()

	m := ModuleBuilder().Controllers(mtPrefixedController{}).Build()
	m.Prefix("v1")

	m.NewModule()

	if len(m.RESTMainHandlers) != 1 {
		t.Fatalf("expected 1 REST main handler, got %d", len(m.RESTMainHandlers))
	}
	h := m.RESTMainHandlers[0]
	if h.Method != "GET" {
		t.Error(test.DiffMessage(h.Method, "GET", "REST main handler method"))
	}
	if !strings.Contains(h.Route, "v1") || !strings.Contains(h.Route, "mtprefixtarget") {
		t.Error(test.DiffMessage(h.Route, "route containing both the module prefix and the resource name", "REST main handler route"))
	}
	if h.Handler == nil {
		t.Error(test.DiffMessage(nil, "non-nil handler", "REST main handler Handler"))
	}
	if h.MainHandlerName != "READ_mtprefixtarget" {
		t.Error(test.DiffMessage(h.MainHandlerName, "READ_mtprefixtarget", "REST main handler name"))
	}
}

type mtGuardPriorityController struct {
	common.REST
	common.Guard
}

func (c mtGuardPriorityController) NewController() Controller {
	c.BindGuard(mtPriorityGuard{Passthrough: mtPassthroughValue{Tag: "original"}})
	return c
}
func (c mtGuardPriorityController) READ_mtguardpriority() string { return "ok" }

func TestNewModule_RESTGuardInjection_PriorityChain(t *testing.T) {
	resetModuleGlobals()
	mtPriorityGuardSeen = mtPriorityGuard{}
	seedPriorityChainGlobals()

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtGuardPriorityController{}).
		Build()

	m.NewModule()

	if len(m.RESTGuards) != 1 {
		t.Fatalf("expected 1 REST guard, got %d", len(m.RESTGuards))
	}
	handler, ok := m.RESTGuards[0].Handler.(func(*ctx.HTTPContext) bool)
	if !ok {
		t.Fatalf("unexpected REST guard handler type %T", m.RESTGuards[0].Handler)
	}
	handler(nil)

	assertPriorityChainSeen(t, "REST guard")
}

type mtGuardUnexportedController struct {
	common.REST
	common.Guard
}

func (c mtGuardUnexportedController) NewController() Controller {
	c.BindGuard(mtUnexportedFieldGuard{})
	return c
}
func (c mtGuardUnexportedController) READ_mtguardunexported() string { return "ok" }

func TestNewModule_RESTGuardInjection_UnexportedFieldPanics(t *testing.T) {
	resetModuleGlobals()
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "guard with an unexported field should panic during injection"))
		}
	}()

	m := ModuleBuilder().Controllers(mtGuardUnexportedController{}).Build()
	m.NewModule()
}

type mtGuardUnresolvedController struct {
	common.REST
	common.Guard
}

func (c mtGuardUnresolvedController) NewController() Controller {
	c.BindGuard(mtUnresolvedFieldGuard{})
	return c
}
func (c mtGuardUnresolvedController) READ_mtguardunresolved() string { return "ok" }

func TestNewModule_RESTGuardInjection_UnresolvedDependencyPanics(t *testing.T) {
	resetModuleGlobals()
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "guard field with an unresolved provider dependency should panic"))
		}
	}()

	m := ModuleBuilder().Controllers(mtGuardUnresolvedController{}).Build()
	m.NewModule()
}

type mtMiddlewareController struct {
	common.REST
	common.Middleware
}

func (c mtMiddlewareController) NewController() Controller {
	c.BindMiddleware(mtSimpleMiddleware{})
	return c
}
func (c mtMiddlewareController) READ_mtmiddleware() string { return "ok" }

func TestNewModule_RESTMiddlewareInjection_LocalProviderRegistered(t *testing.T) {
	resetModuleGlobals()
	mtSimpleMiddlewareSeen = mtSimpleMiddleware{}

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtMiddlewareController{}).
		Build()

	m.NewModule()

	if len(m.RESTMiddlewares) != 1 {
		t.Fatalf("expected 1 REST middleware, got %d", len(m.RESTMiddlewares))
	}
	handler, ok := m.RESTMiddlewares[0].Handler.(func(*http.Request, http.ResponseWriter, ctx.Next))
	if !ok {
		t.Fatalf("unexpected REST middleware handler type %T", m.RESTMiddlewares[0].Handler)
	}
	called := false
	handler(nil, nil, func() { called = true })

	if !called {
		t.Error(test.DiffMessage(false, true, "REST middleware should call next()"))
	}
	if mtSimpleMiddlewareSeen.P.Source != "local" {
		t.Error(test.DiffMessage(mtSimpleMiddlewareSeen.P.Source, "local", "REST middleware field should resolve from local module providers"))
	}
}

type mtInterceptorController struct {
	common.REST
	common.Interceptor
}

func (c mtInterceptorController) NewController() Controller {
	c.BindInterceptor(mtSimpleInterceptor{})
	return c
}
func (c mtInterceptorController) READ_mtinterceptor() string { return "ok" }

func TestNewModule_RESTInterceptorInjection_LocalProviderRegistered(t *testing.T) {
	resetModuleGlobals()
	mtSimpleInterceptorSeen = mtSimpleInterceptor{}

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtInterceptorController{}).
		Build()

	m.NewModule()

	if len(m.RESTInterceptors) != 1 {
		t.Fatalf("expected 1 REST interceptor, got %d", len(m.RESTInterceptors))
	}
	handler, ok := m.RESTInterceptors[0].Handler.(func(*ctx.HTTPContext, *aggregation.Aggregation) any)
	if !ok {
		t.Fatalf("unexpected REST interceptor handler type %T", m.RESTInterceptors[0].Handler)
	}
	handler(nil, aggregation.NewAggregation())

	if mtSimpleInterceptorSeen.P.Source != "local" {
		t.Error(test.DiffMessage(mtSimpleInterceptorSeen.P.Source, "local", "REST interceptor field should resolve from local module providers"))
	}
}

type mtExFilterSingleController struct {
	common.REST
	common.ExceptionFilter
}

func (c mtExFilterSingleController) NewController() Controller {
	c.BindExceptionFilter(mtExFilterOneField{})
	return c
}
func (c mtExFilterSingleController) READ_mtexfiltersingle() string { return "ok" }

func TestNewModule_RESTExceptionFilter_SingleController_Registered(t *testing.T) {
	resetModuleGlobals()
	mtExFilterOneFieldSeen = mtExFilterOneField{}

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtExFilterSingleController{}).
		Build()

	m.NewModule()

	if len(m.RESTExceptionFilters) != 1 {
		t.Fatalf("expected 1 REST exception filter, got %d", len(m.RESTExceptionFilters))
	}
	handler, ok := m.RESTExceptionFilters[0].Handler.(func(*ctx.HTTPContext, *exception.Exception))
	if !ok {
		t.Fatalf("unexpected REST exception filter handler type %T", m.RESTExceptionFilters[0].Handler)
	}
	handler(nil, nil)

	if mtExFilterOneFieldSeen.P.Source != "local" {
		t.Error(test.DiffMessage(mtExFilterOneFieldSeen.P.Source, "local", "exceptionFilter field should resolve from local module providers when the controller index happens to match the field index"))
	}
}

type mtExFilterMultiController0 struct {
	common.REST
	common.ExceptionFilter
}

func (c mtExFilterMultiController0) NewController() Controller {
	c.BindExceptionFilter(mtExFilterOneField{})
	return c
}
func (c mtExFilterMultiController0) READ_mtexfilterbugc0() string { return "ok" }

type mtExFilterMultiController1 struct {
	common.REST
	common.ExceptionFilter
}

func (c mtExFilterMultiController1) NewController() Controller {
	c.BindExceptionFilter(mtExFilterOneField{})
	return c
}
func (c mtExFilterMultiController1) READ_mtexfilterbugc1() string { return "ok" }

// TestNewModule_RESTExceptionFilter_TwoControllers_CorrectFieldIndex used to
// pin a bug in module.go's REST exceptionFilter callback: it discarded its
// own field-index parameter and instead reused the outer
// `for i, controller := range m.controllers` loop variable to index into
// the exceptionFilter's fields, so the 2nd+ controller in a module panicked
// with "reflect: Field index out of range" whenever its exceptionFilter had
// fewer fields than that controller's index. The REST/WS callbacks were
// unified into buildFieldInjectionCallback (fn.go) during the module.go
// refactor, which fixed this as a side effect - both layers now share the
// WS side's (always correct) field-index handling. This test asserts the
// fixed behavior; its WS counterpart just below asserts the same thing for
// WS to guard against a regression re-introducing the split.
func TestNewModule_RESTExceptionFilter_TwoControllers_CorrectFieldIndex(t *testing.T) {
	resetModuleGlobals()

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtExFilterMultiController0{}, mtExFilterMultiController1{}).
		Build()

	m.NewModule()

	if len(m.RESTExceptionFilters) != 2 {
		t.Fatalf("expected 2 REST exception filters, got %d", len(m.RESTExceptionFilters))
	}
	for _, ef := range m.RESTExceptionFilters {
		handler, ok := ef.Handler.(func(*ctx.HTTPContext, *exception.Exception))
		if !ok {
			t.Fatalf("unexpected REST exception filter handler type %T", ef.Handler)
		}
		handler(nil, nil) // must not panic
	}
}

// ---------------------------------------------------------------------------
// WS controller processing
// ---------------------------------------------------------------------------

type mtWSMainController struct{ common.WS }

func (c mtWSMainController) NewController() Controller         { return c }
func (c mtWSMainController) SUBSCRIBE_mtwsmainhandler() string { return "ok" }

func TestNewModule_WSMainHandler_Registered(t *testing.T) {
	resetModuleGlobals()

	m := ModuleBuilder().Controllers(mtWSMainController{}).Build()
	m.NewModule()

	if len(m.WSMainHandlers) != 1 {
		t.Fatalf("expected 1 WS main handler, got %d", len(m.WSMainHandlers))
	}
	if m.WSMainHandlers[0].EventName != "mtwsmainhandler" {
		t.Error(test.DiffMessage(m.WSMainHandlers[0].EventName, "mtwsmainhandler", "WS main handler event name"))
	}
	if m.WSMainHandlers[0].Handler == nil {
		t.Error(test.DiffMessage(nil, "non-nil handler", "WS main handler Handler"))
	}
}

type mtPriorityGuardWS struct {
	Local       mtLocalProvider
	GlobalOnly  mtGlobalOnlyProvider
	IfaceOnly   mtInterfaceOnlyProvider
	Passthrough mtPassthroughValue
}

var mtPriorityGuardWSSeen mtPriorityGuardWS

func (g mtPriorityGuardWS) CanActivate(_ *ctx.WSContext) bool {
	mtPriorityGuardWSSeen = g
	return true
}

func assertPriorityChainSeenWS(t *testing.T, label string) {
	t.Helper()
	if mtPriorityGuardWSSeen.Local.Source != "local" {
		t.Error(test.DiffMessage(mtPriorityGuardWSSeen.Local.Source, "local", label+": local module provider must win over a global provider of the same type"))
	}
	if mtPriorityGuardWSSeen.GlobalOnly.Source != "global" {
		t.Error(test.DiffMessage(mtPriorityGuardWSSeen.GlobalOnly.Source, "global", label+": field with no local provider must fall back to globalProviderByKey"))
	}
	if mtPriorityGuardWSSeen.IfaceOnly.Source != "interface" {
		t.Error(test.DiffMessage(mtPriorityGuardWSSeen.IfaceOnly.Source, "interface", label+": field with no local/global provider must fall back to globalInterfaceByKey"))
	}
	if mtPriorityGuardWSSeen.Passthrough.Tag != "original" {
		t.Error(test.DiffMessage(mtPriorityGuardWSSeen.Passthrough.Tag, "original", label+": non-Provider field must pass through the bound instance's original value"))
	}
}

type mtWSGuardPriorityController struct {
	common.WS
	common.Guard
}

func (c mtWSGuardPriorityController) NewController() Controller {
	c.BindGuard(mtPriorityGuardWS{Passthrough: mtPassthroughValue{Tag: "original"}})
	return c
}
func (c mtWSGuardPriorityController) SUBSCRIBE_mtwsguardpriority() string { return "ok" }

func TestNewModule_WSGuardInjection_PriorityChain(t *testing.T) {
	resetModuleGlobals()
	mtPriorityGuardWSSeen = mtPriorityGuardWS{}
	seedPriorityChainGlobals()

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtWSGuardPriorityController{}).
		Build()

	m.NewModule()

	if len(m.WSGuards) != 1 {
		t.Fatalf("expected 1 WS guard, got %d", len(m.WSGuards))
	}
	handler, ok := m.WSGuards[0].Handler.(func(*ctx.WSContext) bool)
	if !ok {
		t.Fatalf("unexpected WS guard handler type %T", m.WSGuards[0].Handler)
	}
	handler(nil)

	assertPriorityChainSeenWS(t, "WS guard")
}

type mtWSGuardUnexportedController struct {
	common.WS
	common.Guard
}

func (c mtWSGuardUnexportedController) NewController() Controller {
	c.BindGuard(mtUnexportedFieldGuard{})
	return c
}
func (c mtWSGuardUnexportedController) SUBSCRIBE_mtwsguardunexported() string { return "ok" }

func TestNewModule_WSGuardInjection_UnexportedFieldPanics(t *testing.T) {
	resetModuleGlobals()
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "WS guard with an unexported field should panic during injection"))
		}
	}()

	m := ModuleBuilder().Controllers(mtWSGuardUnexportedController{}).Build()
	m.NewModule()
}

type mtWSGuardUnresolvedController struct {
	common.WS
	common.Guard
}

func (c mtWSGuardUnresolvedController) NewController() Controller {
	c.BindGuard(mtUnresolvedFieldGuard{})
	return c
}
func (c mtWSGuardUnresolvedController) SUBSCRIBE_mtwsguardunresolved() string { return "ok" }

func TestNewModule_WSGuardInjection_UnresolvedDependencyPanics(t *testing.T) {
	resetModuleGlobals()
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "WS guard field with an unresolved provider dependency should panic"))
		}
	}()

	m := ModuleBuilder().Controllers(mtWSGuardUnresolvedController{}).Build()
	m.NewModule()
}

type mtSimpleInterceptorWS struct{ P mtLocalProvider }

var mtSimpleInterceptorWSSeen mtSimpleInterceptorWS

func (ic mtSimpleInterceptorWS) Intercept(_ *ctx.WSContext, _ *aggregation.Aggregation) any {
	mtSimpleInterceptorWSSeen = ic
	return nil
}

type mtWSInterceptorController struct {
	common.WS
	common.Interceptor
}

func (c mtWSInterceptorController) NewController() Controller {
	c.BindInterceptor(mtSimpleInterceptorWS{})
	return c
}
func (c mtWSInterceptorController) SUBSCRIBE_mtwsinterceptor() string { return "ok" }

func TestNewModule_WSInterceptorInjection_LocalProviderRegistered(t *testing.T) {
	resetModuleGlobals()
	mtSimpleInterceptorWSSeen = mtSimpleInterceptorWS{}

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtWSInterceptorController{}).
		Build()

	m.NewModule()

	if len(m.WSInterceptors) != 1 {
		t.Fatalf("expected 1 WS interceptor, got %d", len(m.WSInterceptors))
	}
	handler, ok := m.WSInterceptors[0].Handler.(func(*ctx.WSContext, *aggregation.Aggregation) any)
	if !ok {
		t.Fatalf("unexpected WS interceptor handler type %T", m.WSInterceptors[0].Handler)
	}
	handler(nil, aggregation.NewAggregation())

	if mtSimpleInterceptorWSSeen.P.Source != "local" {
		t.Error(test.DiffMessage(mtSimpleInterceptorWSSeen.P.Source, "local", "WS interceptor field should resolve from local module providers"))
	}
}

type mtExFilterOneFieldWS struct{ P mtLocalProvider }

var mtExFilterOneFieldWSSeen mtExFilterOneFieldWS

func (e mtExFilterOneFieldWS) Catch(_ *ctx.WSContext, _ *exception.Exception) {
	mtExFilterOneFieldWSSeen = e
}

type mtWSExFilterController0 struct {
	common.WS
	common.ExceptionFilter
}

func (c mtWSExFilterController0) NewController() Controller {
	c.BindExceptionFilter(mtExFilterOneFieldWS{})
	return c
}
func (c mtWSExFilterController0) SUBSCRIBE_mtwsexfilterc0() string { return "ok" }

type mtWSExFilterController1 struct {
	common.WS
	common.ExceptionFilter
}

func (c mtWSExFilterController1) NewController() Controller {
	c.BindExceptionFilter(mtExFilterOneFieldWS{})
	return c
}
func (c mtWSExFilterController1) SUBSCRIBE_mtwsexfilterc1() string { return "ok" }

// TestNewModule_WSExceptionFilter_TwoControllers_CorrectFieldIndex confirms
// the WS exceptionFilter callback does NOT have the REST field-index bug
// (see TestNewModule_RESTExceptionFilter_TwoControllers_FieldIndexBug): a
// 2nd controller with a 1-field exceptionFilter must inject correctly
// instead of panicking. Any refactor that unifies the REST/WS exceptionFilter
// callbacks into one helper must preserve THIS behavior, not the REST one.
func TestNewModule_WSExceptionFilter_TwoControllers_CorrectFieldIndex(t *testing.T) {
	resetModuleGlobals()

	m := ModuleBuilder().
		Providers(mtLocalProvider{}).
		Controllers(mtWSExFilterController0{}, mtWSExFilterController1{}).
		Build()

	m.NewModule()

	if len(m.WSExceptionFilters) != 2 {
		t.Fatalf("expected 2 WS exception filters, got %d", len(m.WSExceptionFilters))
	}
	for _, ef := range m.WSExceptionFilters {
		handler, ok := ef.Handler.(func(*ctx.WSContext, *exception.Exception))
		if !ok {
			t.Fatalf("unexpected WS exception filter handler type %T", ef.Handler)
		}
		handler(nil, nil) // must not panic
	}
}

// ---------------------------------------------------------------------------
// concurrency: NewModule mutates package-level state (mainModulePtr,
// globalProviderByKey, providerSingletonByKey, staticModuleByDynamicPtr,
// globalPrefixesByController) on top of the per-Module mutex, so two independent module
// trees built and initialized concurrently (e.g. two apps in the same test
// binary) must not corrupt that shared state.
// ---------------------------------------------------------------------------

type mtConcurrentProvider struct{}

func (p mtConcurrentProvider) NewProvider() Provider { return p }

type mtConcurrentController struct{ common.REST }

func (c mtConcurrentController) NewController() Controller         { return c }
func (c mtConcurrentController) READ_mtconcurrentresource() string { return "ok" }

func TestNewModule_ConcurrentInvocation_NoDataRace(t *testing.T) {
	resetModuleGlobals()

	const n = 32
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()

			m := ModuleBuilder().
				Providers(mtConcurrentProvider{}).
				Controllers(mtConcurrentController{}).
				Build()
			m.NewModule()
		}()
	}
	wg.Wait()
}
