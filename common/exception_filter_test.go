package common

import (
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/routing"
)

type mockExFilter struct{}

func (mockExFilter) Catch(_ *ctx.Context, _ *exception.Exception) {}

var noopCB = func(_ int, _ reflect.Type, _ reflect.Value, _ reflect.Value) {}

func buildREST(fnToRoute map[string]string) *REST {
	r := &REST{
		PatternToFuncNameMap: make(map[string]string, len(fnToRoute)),
		FuncNameToPatternMap: make(map[string]string, len(fnToRoute)),
	}
	for fn, route := range fnToRoute {
		p := routing.MethodRouteVersionToPattern("GET", route, "")
		r.PatternToFuncNameMap[p] = fn
		r.FuncNameToPatternMap[fn] = p
	}
	return r
}

func buildWS(patternToFn map[string]string) *WS {
	ws := &WS{
		funcNameByEvent: make(map[string]string, len(patternToFn)),
	}
	for p, fn := range patternToFn {
		ws.funcNameByEvent[p] = fn
	}
	return ws
}

func TestBindExceptionFilter_Chaining(t *testing.T) {
	e := &ExceptionFilter{}
	ret := e.BindExceptionFilter(mockExFilter{})
	if ret != e {
		t.Error(test.DiffMessage(ret, e, "BindExceptionFilter should return self"))
	}
	if len(e.ExceptionFilterHandlers) != 1 {
		t.Error(test.DiffMessage(len(e.ExceptionFilterHandlers), 1, "one handler after one bind"))
	}
	e.BindExceptionFilter(mockExFilter{})
	if len(e.ExceptionFilterHandlers) != 2 {
		t.Error(test.DiffMessage(len(e.ExceptionFilterHandlers), 2, "two handlers after two binds"))
	}
}

func TestInjectProvidersIntoRESTExceptionFilters_Empty(t *testing.T) {
	e := &ExceptionFilter{}
	r := buildREST(map[string]string{"READ_users": "/users/"})

	items := e.InjectProvidersIntoRESTExceptionFilters(r, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound filters → empty result"))
	}
}

func TestInjectProvidersIntoRESTExceptionFilters_ApplyAll(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(mockExFilter{})

	r := buildREST(map[string]string{
		"READ_users":    "/users/",
		"CREATE_orders": "/orders/",
	})

	items := e.InjectProvidersIntoRESTExceptionFilters(r, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "filter with no handlers applies to all patterns"))
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

func TestInjectProvidersIntoRESTExceptionFilters_MainHandlerName(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(mockExFilter{})

	r := buildREST(map[string]string{"READ_items": "/items/"})
	items := e.InjectProvidersIntoRESTExceptionFilters(r, noopCB)

	if len(items) != 1 {
		t.Error(test.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].REST.Common.MainHandlerName != "READ_items" {
		t.Error(test.DiffMessage(items[0].REST.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoWSExceptionFilters_Empty(t *testing.T) {
	e := &ExceptionFilter{}
	ws := buildWS(map[string]string{"message": "ON_message"})

	items := e.InjectProvidersIntoWSExceptionFilters(ws, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound filters → empty result"))
	}
}

func TestInjectProvidersIntoWSExceptionFilters_ApplyAll(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(mockExFilter{})

	ws := buildWS(map[string]string{
		"message": "ON_message",
		"status":  "ON_status",
	})

	items := e.InjectProvidersIntoWSExceptionFilters(ws, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "filter with no handlers applies to all WS patterns"))
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
