package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func TestInjectProvidersIntoHTTPGuards_Empty(t *testing.T) {
	g := &Guard{}
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	items := g.InjectProvidersIntoHTTPGuards(r, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound guards → empty result"))
	}
}

func TestInjectProvidersIntoHTTPGuards_ApplyAll(t *testing.T) {
	g := &Guard{}
	g.BindGuard(mockGuarder{})

	r := buildHTTP(map[string]string{
		"READ_users":    "/users/",
		"CREATE_orders": "/orders/",
	})

	items := g.InjectProvidersIntoHTTPGuards(r, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "guard with no handlers applies to all patterns"))
	}
	for _, item := range items {
		if item.HTTP.Method != "GET" {
			t.Error(test.DiffMessage(item.HTTP.Method, "GET", "method"))
		}
		if item.HTTP.Pattern == "" {
			t.Error(test.DiffMessage(item.HTTP.Pattern, "non-empty", "pattern"))
		}
		if item.HTTP.Common.Name == "" {
			t.Error(test.DiffMessage(item.HTTP.Common.Name, "non-empty", "name"))
		}
	}
}

func TestInjectProvidersIntoHTTPGuards_MainHandlerName(t *testing.T) {
	g := &Guard{}
	g.BindGuard(mockGuarder{})

	r := buildHTTP(map[string]string{"READ_items": "/items/"})
	items := g.InjectProvidersIntoHTTPGuards(r, noopCB)

	if len(items) != 1 {
		t.Error(test.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].HTTP.Common.MainHandlerName != "READ_items" {
		t.Error(test.DiffMessage(items[0].HTTP.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoHTTPGuards_HandlerIsCallableCanActivate(t *testing.T) {
	g := &Guard{}
	g.BindGuard(denyGuarder{})

	r := buildHTTP(map[string]string{"READ_users": "/users/"})
	items := g.InjectProvidersIntoHTTPGuards(r, noopCB)
	if len(items) != 1 {
		t.Fatal(test.DiffMessage(len(items), 1, "one pattern → one item"))
	}

	canActivate, ok := items[0].HTTP.Common.Handler.(HTTPCanActivate)
	if !ok {
		t.Fatal(test.DiffMessage(items[0].HTTP.Common.Handler, "HTTPCanActivate", "Handler must be callable as HTTPCanActivate"))
	}
	if canActivate(nil) != false {
		t.Error(test.DiffMessage(true, false, "Handler must call through to the bound guard's CanActivate"))
	}
}

func TestInjectProvidersIntoHTTPGuards_NoCanActivate_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(noCanActivateGuarder{})
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "guard with no CanActivate method must panic"))
		}
	}()
	g.InjectProvidersIntoHTTPGuards(r, noopCB)
}

func TestInjectProvidersIntoHTTPGuards_WrongReturnType_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(wrongReturnGuarder{})
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "CanActivate returning non-bool must panic"))
		}
	}()
	g.InjectProvidersIntoHTTPGuards(r, noopCB)
}

func TestInjectProvidersIntoHTTPGuards_WrongParamType_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(wrongParamGuarder{})
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "CanActivate with a non-context param must panic"))
		}
	}()
	g.InjectProvidersIntoHTTPGuards(r, noopCB)
}

func TestAsHTTPGuard_Valid(t *testing.T) {
	fn, ok := AsHTTPGuard(mockGuarder{})
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "mockGuarder must match HTTPCanActivate"))
	}
	if fn(nil) != true {
		t.Error(test.DiffMessage(false, true, "returned fn must call through to CanActivate"))
	}
}

func TestAsHTTPGuard_NoMethod(t *testing.T) {
	_, ok := AsHTTPGuard(noCanActivateGuarder{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a guarder with no CanActivate must not match"))
	}
}

func TestAsHTTPGuard_WrongShape(t *testing.T) {
	_, ok := AsHTTPGuard(wrongReturnGuarder{})
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
