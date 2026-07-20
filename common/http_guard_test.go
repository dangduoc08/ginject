package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func TestInjectProvidersIntoRESTGuards_Empty(t *testing.T) {
	g := &Guard{}
	r := buildREST(map[string]string{"READ_users": "/users/"})

	items := g.InjectProvidersIntoRESTGuards(r, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound guards → empty result"))
	}
}

func TestInjectProvidersIntoRESTGuards_ApplyAll(t *testing.T) {
	g := &Guard{}
	g.BindGuard(mockGuarder{})

	r := buildREST(map[string]string{
		"READ_users":    "/users/",
		"CREATE_orders": "/orders/",
	})

	items := g.InjectProvidersIntoRESTGuards(r, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "guard with no handlers applies to all patterns"))
	}
	for _, item := range items {
		if item.REST.Method != "GET" {
			t.Error(test.DiffMessage(item.REST.Method, "GET", "method"))
		}
		if item.REST.Pattern == "" {
			t.Error(test.DiffMessage(item.REST.Pattern, "non-empty", "pattern"))
		}
		if item.REST.Common.Name == "" {
			t.Error(test.DiffMessage(item.REST.Common.Name, "non-empty", "name"))
		}
	}
}

func TestInjectProvidersIntoRESTGuards_MainHandlerName(t *testing.T) {
	g := &Guard{}
	g.BindGuard(mockGuarder{})

	r := buildREST(map[string]string{"READ_items": "/items/"})
	items := g.InjectProvidersIntoRESTGuards(r, noopCB)

	if len(items) != 1 {
		t.Error(test.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].REST.Common.MainHandlerName != "READ_items" {
		t.Error(test.DiffMessage(items[0].REST.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoRESTGuards_HandlerIsCallableCanActivate(t *testing.T) {
	g := &Guard{}
	g.BindGuard(denyGuarder{})

	r := buildREST(map[string]string{"READ_users": "/users/"})
	items := g.InjectProvidersIntoRESTGuards(r, noopCB)
	if len(items) != 1 {
		t.Fatal(test.DiffMessage(len(items), 1, "one pattern → one item"))
	}

	canActivate, ok := items[0].REST.Common.Handler.(RESTCanActivate)
	if !ok {
		t.Fatal(test.DiffMessage(items[0].REST.Common.Handler, "RESTCanActivate", "Handler must be callable as RESTCanActivate"))
	}
	if canActivate(nil) != false {
		t.Error(test.DiffMessage(true, false, "Handler must call through to the bound guard's CanActivate"))
	}
}

func TestInjectProvidersIntoRESTGuards_NoCanActivate_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(noCanActivateGuarder{})
	r := buildREST(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "guard with no CanActivate method must panic"))
		}
	}()
	g.InjectProvidersIntoRESTGuards(r, noopCB)
}

func TestInjectProvidersIntoRESTGuards_WrongReturnType_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(wrongReturnGuarder{})
	r := buildREST(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "CanActivate returning non-bool must panic"))
		}
	}()
	g.InjectProvidersIntoRESTGuards(r, noopCB)
}

func TestInjectProvidersIntoRESTGuards_WrongParamType_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(wrongParamGuarder{})
	r := buildREST(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "CanActivate with a non-context param must panic"))
		}
	}()
	g.InjectProvidersIntoRESTGuards(r, noopCB)
}

func TestAsRESTGuard_Valid(t *testing.T) {
	fn, ok := AsRESTGuard(mockGuarder{})
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "mockGuarder must match RESTCanActivate"))
	}
	if fn(nil) != true {
		t.Error(test.DiffMessage(false, true, "returned fn must call through to CanActivate"))
	}
}

func TestAsRESTGuard_NoMethod(t *testing.T) {
	_, ok := AsRESTGuard(noCanActivateGuarder{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a guarder with no CanActivate must not match"))
	}
}

func TestAsRESTGuard_WrongShape(t *testing.T) {
	_, ok := AsRESTGuard(wrongReturnGuarder{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a CanActivate returning non-bool must not match"))
	}
}

func TestHandleHTTPGuard_PanicOnDenied(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(r, "non-nil panic", "handleHTTPGuard(nil, false) should panic"))
		}
	}()
	handleHTTPGuard(nil, false)
}

func TestHandleHTTPGuard_CallsNext(t *testing.T) {
	called := false
	c := &ctx.HTTPContext{}
	c.Next = func() { called = true }
	handleHTTPGuard(c, true)
	if !called {
		t.Error(test.DiffMessage(called, true, "handleHTTPGuard should call Next when access is allowed"))
	}
}

func TestBuildHTTPGuardMiddleware_AllowedCallsNext(t *testing.T) {
	called := false
	c := &ctx.HTTPContext{}
	c.Next = func() { called = true }

	mw := BuildHTTPGuardMiddleware(func(*ctx.HTTPContext) bool { return true })
	mw(c)

	if !called {
		t.Error(test.DiffMessage(called, true, "an allowing guard must call Next"))
	}
}

func TestBuildHTTPGuardMiddleware_DeniedPanics(t *testing.T) {
	c := &ctx.HTTPContext{}
	mw := BuildHTTPGuardMiddleware(func(*ctx.HTTPContext) bool { return false })

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "a denying guard must panic"))
		}
	}()
	mw(c)
}
