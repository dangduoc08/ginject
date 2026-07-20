package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func TestInjectProvidersIntoWSGuards_Empty(t *testing.T) {
	g := &Guard{}
	ws := buildWS(map[string]string{"message": "ON_message"})

	items := g.InjectProvidersIntoWSGuards(ws, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound guards → empty result"))
	}
}

func TestInjectProvidersIntoWSGuards_ApplyAll(t *testing.T) {
	g := &Guard{}
	g.BindGuard(mockWSGuarder{})

	ws := buildWS(map[string]string{
		"message": "ON_message",
		"status":  "ON_status",
	})

	items := g.InjectProvidersIntoWSGuards(ws, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "guard with no handlers applies to all WS patterns"))
	}
	for _, item := range items {
		if item.WS.EventName == "" {
			t.Error(test.DiffMessage(item.WS.EventName, "non-empty", "event name"))
		}
		if item.WS.Common.Name == "" {
			t.Error(test.DiffMessage(item.WS.Common.Name, "non-empty", "name"))
		}
	}
}

func TestInjectProvidersIntoWSGuards_HandlerIsCallableCanActivate(t *testing.T) {
	g := &Guard{}
	g.BindGuard(denyWSGuarder{})

	ws := buildWS(map[string]string{"message": "ON_message"})
	items := g.InjectProvidersIntoWSGuards(ws, noopCB)
	if len(items) != 1 {
		t.Fatal(test.DiffMessage(len(items), 1, "one pattern → one item"))
	}

	canActivate, ok := items[0].WS.Common.Handler.(WSCanActivate)
	if !ok {
		t.Fatal(test.DiffMessage(items[0].WS.Common.Handler, "WSCanActivate", "Handler must be callable as WSCanActivate"))
	}
	if canActivate(nil) != false {
		t.Error(test.DiffMessage(true, false, "Handler must call through to the bound guard's CanActivate"))
	}
}

func TestInjectProvidersIntoWSGuards_NoCanActivate_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(noCanActivateGuarder{})
	ws := buildWS(map[string]string{"message": "ON_message"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "guard with no CanActivate method must panic"))
		}
	}()
	g.InjectProvidersIntoWSGuards(ws, noopCB)
}

func TestInjectProvidersIntoWSGuards_WrongReturnType_Panics(t *testing.T) {
	g := &Guard{}
	g.BindGuard(wrongReturnGuarder{})
	ws := buildWS(map[string]string{"message": "ON_message"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "CanActivate returning non-bool must panic"))
		}
	}()
	g.InjectProvidersIntoWSGuards(ws, noopCB)
}

func TestAsWSGuard_Valid(t *testing.T) {
	fn, ok := AsWSGuard(mockWSGuarder{})
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "mockWSGuarder must match WSCanActivate"))
	}
	if fn(nil) != true {
		t.Error(test.DiffMessage(false, true, "returned fn must call through to CanActivate"))
	}
}

func TestAsWSGuard_NoMethod(t *testing.T) {
	_, ok := AsWSGuard(noCanActivateGuarder{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a guarder with no CanActivate must not match"))
	}
}

func TestAsWSGuard_WrongShape(t *testing.T) {
	_, ok := AsWSGuard(wrongParamGuarder{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a CanActivate with a non-context param must not match"))
	}
}

func TestHandleWSGuard_PanicOnDenied(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(r, "non-nil panic", "handleWSGuard(nil, false) should panic"))
		}
	}()
	handleWSGuard(nil, false)
}

func TestHandleWSGuard_CallsNext(t *testing.T) {
	called := false
	c := &ctx.WSContext{}
	c.Next = func() { called = true }
	handleWSGuard(c, true)
	if !called {
		t.Error(test.DiffMessage(called, true, "handleWSGuard should call Next when access is allowed"))
	}
}

func TestBuildWSGuardMiddleware_AllowedCallsNext(t *testing.T) {
	called := false
	c := &ctx.WSContext{}
	c.Next = func() { called = true }

	mw := BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return true })
	mw(c)

	if !called {
		t.Error(test.DiffMessage(called, true, "an allowing guard must call Next"))
	}
}

func TestBuildWSGuardMiddleware_DeniedPanics(t *testing.T) {
	c := &ctx.WSContext{}
	mw := BuildWSGuardMiddleware(func(*ctx.WSContext) bool { return false })

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "a denying guard must panic"))
		}
	}()
	mw(c)
}
