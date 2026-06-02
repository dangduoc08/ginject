package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

type mockGuarder struct{}

func (mockGuarder) CanActivate(_ *ctx.Context) bool { return true }

func TestBindGuard_Chaining(t *testing.T) {
	g := &Guard{}
	ret := g.BindGuard(mockGuarder{})
	if ret != g {
		t.Error(testutils.DiffMessage(ret, g, "BindGuard should return self"))
	}
	if len(g.GuardHandlers) != 1 {
		t.Error(testutils.DiffMessage(len(g.GuardHandlers), 1, "one handler after one bind"))
	}
	g.BindGuard(mockGuarder{})
	if len(g.GuardHandlers) != 2 {
		t.Error(testutils.DiffMessage(len(g.GuardHandlers), 2, "two handlers after two binds"))
	}
}

func TestInjectProvidersIntoRESTGuards_Empty(t *testing.T) {
	g := &Guard{}
	r := buildREST(map[string]string{"READ_users": "/users/"})

	items := g.InjectProvidersIntoRESTGuards(r, noopCB)
	if len(items) != 0 {
		t.Error(testutils.DiffMessage(len(items), 0, "no bound guards → empty result"))
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
		t.Error(testutils.DiffMessage(len(items), 2, "guard with no handlers applies to all patterns"))
	}
	for _, item := range items {
		if item.REST.Method != "GET" {
			t.Error(testutils.DiffMessage(item.REST.Method, "GET", "method"))
		}
		if item.REST.Pattern == "" {
			t.Error(testutils.DiffMessage(item.REST.Pattern, "non-empty", "pattern"))
		}
		if item.REST.Common.Name == "" {
			t.Error(testutils.DiffMessage(item.REST.Common.Name, "non-empty", "name"))
		}
	}
}

func TestInjectProvidersIntoRESTGuards_MainHandlerName(t *testing.T) {
	g := &Guard{}
	g.BindGuard(mockGuarder{})

	r := buildREST(map[string]string{"READ_items": "/items/"})
	items := g.InjectProvidersIntoRESTGuards(r, noopCB)

	if len(items) != 1 {
		t.Error(testutils.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].REST.Common.MainHandlerName != "READ_items" {
		t.Error(testutils.DiffMessage(items[0].REST.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoWSGuards_Empty(t *testing.T) {
	g := &Guard{}
	ws := buildWS(map[string]string{"message": "ON_message"})

	items := g.InjectProvidersIntoWSGuards(ws, noopCB)
	if len(items) != 0 {
		t.Error(testutils.DiffMessage(len(items), 0, "no bound guards → empty result"))
	}
}

func TestInjectProvidersIntoWSGuards_ApplyAll(t *testing.T) {
	g := &Guard{}
	g.BindGuard(mockGuarder{})

	ws := buildWS(map[string]string{
		"message": "ON_message",
		"status":  "ON_status",
	})

	items := g.InjectProvidersIntoWSGuards(ws, noopCB)
	if len(items) != 2 {
		t.Error(testutils.DiffMessage(len(items), 2, "guard with no handlers applies to all WS patterns"))
	}
	for _, item := range items {
		if item.WS.EventName == "" {
			t.Error(testutils.DiffMessage(item.WS.EventName, "non-empty", "event name"))
		}
		if item.WS.Common.Name == "" {
			t.Error(testutils.DiffMessage(item.WS.Common.Name, "non-empty", "name"))
		}
	}
}
