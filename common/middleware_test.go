package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

type mockMiddlewareFn struct{}

func (mockMiddlewareFn) Use(_ *ctx.Context, _ ctx.Next) {}

func TestBindMiddleware_Chaining(t *testing.T) {
	m := &Middleware{}
	ret := m.BindMiddleware(mockMiddlewareFn{})
	if ret != m {
		t.Error(testutils.DiffMessage(ret, m, "BindMiddleware should return self"))
	}
	if len(m.MiddlewareHandlers) != 1 {
		t.Error(testutils.DiffMessage(len(m.MiddlewareHandlers), 1, "one handler after one bind"))
	}
	m.BindMiddleware(mockMiddlewareFn{})
	if len(m.MiddlewareHandlers) != 2 {
		t.Error(testutils.DiffMessage(len(m.MiddlewareHandlers), 2, "two handlers after two binds"))
	}
}

func TestInjectProvidersIntoRESTMiddlewares_Empty(t *testing.T) {
	m := &Middleware{}
	r := buildREST(map[string]string{"READ_users": "/users/"})

	items := m.InjectProvidersIntoRESTMiddlewares(r, noopCB)
	if len(items) != 0 {
		t.Error(testutils.DiffMessage(len(items), 0, "no bound middlewares → empty result"))
	}
}

func TestInjectProvidersIntoRESTMiddlewares_ApplyAll(t *testing.T) {
	m := &Middleware{}
	m.BindMiddleware(mockMiddlewareFn{})

	r := buildREST(map[string]string{
		"READ_users":    "/users/",
		"CREATE_orders": "/orders/",
	})

	items := m.InjectProvidersIntoRESTMiddlewares(r, noopCB)
	if len(items) != 2 {
		t.Error(testutils.DiffMessage(len(items), 2, "middleware with no handlers applies to all patterns"))
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

func TestInjectProvidersIntoRESTMiddlewares_MainHandlerName(t *testing.T) {
	m := &Middleware{}
	m.BindMiddleware(mockMiddlewareFn{})

	r := buildREST(map[string]string{"READ_items": "/items/"})
	items := m.InjectProvidersIntoRESTMiddlewares(r, noopCB)

	if len(items) != 1 {
		t.Error(testutils.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].REST.Common.MainHandlerName != "READ_items" {
		t.Error(testutils.DiffMessage(items[0].REST.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoWSMiddlewares_Empty(t *testing.T) {
	m := &Middleware{}
	ws := buildWS("chat", map[string]string{"chat_/message/": "ON_message"})

	items := m.InjectProvidersIntoWSMiddlewares(ws, noopCB)
	if len(items) != 0 {
		t.Error(testutils.DiffMessage(len(items), 0, "no bound middlewares → empty result"))
	}
}

func TestInjectProvidersIntoWSMiddlewares_ApplyAll(t *testing.T) {
	m := &Middleware{}
	m.BindMiddleware(mockMiddlewareFn{})

	ws := buildWS("chat", map[string]string{
		"chat_/message/": "ON_message",
		"chat_/status/":  "ON_status",
	})

	items := m.InjectProvidersIntoWSMiddlewares(ws, noopCB)
	if len(items) != 2 {
		t.Error(testutils.DiffMessage(len(items), 2, "middleware with no handlers applies to all WS patterns"))
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
