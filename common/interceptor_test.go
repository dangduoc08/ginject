package common

import (
	"testing"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/testutils"
)

type mockInterceptable struct{}

func (mockInterceptable) Intercept(_ *ctx.Context, _ *aggregation.Aggregation) any { return nil }

func TestBindInterceptor_Chaining(t *testing.T) {
	i := &Interceptor{}
	ret := i.BindInterceptor(mockInterceptable{})
	if ret != i {
		t.Error(testutils.DiffMessage(ret, i, "BindInterceptor should return self"))
	}
	if len(i.InterceptorHandlers) != 1 {
		t.Error(testutils.DiffMessage(len(i.InterceptorHandlers), 1, "one handler after one bind"))
	}
	i.BindInterceptor(mockInterceptable{})
	if len(i.InterceptorHandlers) != 2 {
		t.Error(testutils.DiffMessage(len(i.InterceptorHandlers), 2, "two handlers after two binds"))
	}
}

func TestInjectProvidersIntoRESTInterceptors_Empty(t *testing.T) {
	ic := &Interceptor{}
	r := buildREST(map[string]string{"READ_users": "/users/"})

	items := ic.InjectProvidersIntoRESTInterceptors(r, noopCB)
	if len(items) != 0 {
		t.Error(testutils.DiffMessage(len(items), 0, "no bound interceptors → empty result"))
	}
}

func TestInjectProvidersIntoRESTInterceptors_ApplyAll(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockInterceptable{})

	r := buildREST(map[string]string{
		"READ_users":    "/users/",
		"CREATE_orders": "/orders/",
	})

	items := ic.InjectProvidersIntoRESTInterceptors(r, noopCB)
	if len(items) != 2 {
		t.Error(testutils.DiffMessage(len(items), 2, "interceptor with no handlers applies to all patterns"))
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

func TestInjectProvidersIntoRESTInterceptors_MainHandlerName(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockInterceptable{})

	r := buildREST(map[string]string{"READ_items": "/items/"})
	items := ic.InjectProvidersIntoRESTInterceptors(r, noopCB)

	if len(items) != 1 {
		t.Error(testutils.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].REST.Common.MainHandlerName != "READ_items" {
		t.Error(testutils.DiffMessage(items[0].REST.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoWSInterceptors_Empty(t *testing.T) {
	ic := &Interceptor{}
	ws := buildWS("chat", map[string]string{"chat_/message/": "ON_message"})

	items := ic.InjectProvidersIntoWSInterceptors(ws, noopCB)
	if len(items) != 0 {
		t.Error(testutils.DiffMessage(len(items), 0, "no bound interceptors → empty result"))
	}
}

func TestInjectProvidersIntoWSInterceptors_ApplyAll(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockInterceptable{})

	ws := buildWS("chat", map[string]string{
		"chat_/message/": "ON_message",
		"chat_/status/":  "ON_status",
	})

	items := ic.InjectProvidersIntoWSInterceptors(ws, noopCB)
	if len(items) != 2 {
		t.Error(testutils.DiffMessage(len(items), 2, "interceptor with no handlers applies to all WS patterns"))
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
