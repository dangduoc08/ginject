package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func newInterceptorTestContext() *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	c.Init(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	return c
}

func TestInjectProvidersIntoHTTPInterceptors_Empty(t *testing.T) {
	ic := &Interceptor{}
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	items := ic.InjectProvidersIntoHTTPInterceptors(r, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound interceptors → empty result"))
	}
}

func TestInjectProvidersIntoHTTPInterceptors_ApplyAll(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockInterceptable{})

	r := buildHTTP(map[string]string{
		"READ_users":    "/users/",
		"CREATE_orders": "/orders/",
	})

	items := ic.InjectProvidersIntoHTTPInterceptors(r, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "interceptor with no handlers applies to all patterns"))
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

func TestInjectProvidersIntoHTTPInterceptors_MainHandlerName(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockInterceptable{})

	r := buildHTTP(map[string]string{"READ_items": "/items/"})
	items := ic.InjectProvidersIntoHTTPInterceptors(r, noopCB)

	if len(items) != 1 {
		t.Error(test.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].HTTP.Common.MainHandlerName != "READ_items" {
		t.Error(test.DiffMessage(items[0].HTTP.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoHTTPInterceptors_HandlerIsCallableIntercept(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockInterceptable{})

	r := buildHTTP(map[string]string{"READ_users": "/users/"})
	items := ic.InjectProvidersIntoHTTPInterceptors(r, noopCB)
	if len(items) != 1 {
		t.Fatal(test.DiffMessage(len(items), 1, "one pattern → one item"))
	}

	if _, ok := items[0].HTTP.Common.Handler.(HTTPIntercept); !ok {
		t.Fatal(test.DiffMessage(items[0].HTTP.Common.Handler, "HTTPIntercept", "Handler must be callable as HTTPIntercept"))
	}
}

func TestInjectProvidersIntoHTTPInterceptors_NoIntercept_Panics(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(noInterceptMethod{})
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "interceptor with no Intercept method must panic"))
		}
	}()
	ic.InjectProvidersIntoHTTPInterceptors(r, noopCB)
}

func TestInjectProvidersIntoHTTPInterceptors_WrongParamType_Panics(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(wrongParamInterceptable{})
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "Intercept with a non-context first param must panic"))
		}
	}()
	ic.InjectProvidersIntoHTTPInterceptors(r, noopCB)
}

func TestAsHTTPInterceptor_Valid(t *testing.T) {
	fn, ok := AsHTTPInterceptor(mockInterceptable{})
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "mockInterceptable must match HTTPIntercept"))
	}
	fn(nil, aggregation.NewAggregation())
}

func TestAsHTTPInterceptor_NoMethod(t *testing.T) {
	_, ok := AsHTTPInterceptor(noInterceptMethod{})
	if ok {
		t.Error(test.DiffMessage(true, false, "an interceptable with no Intercept must not match"))
	}
}

func TestAsHTTPInterceptor_WrongShape(t *testing.T) {
	_, ok := AsHTTPInterceptor(wrongParamInterceptable{})
	if ok {
		t.Error(test.DiffMessage(true, false, "an Intercept with a non-context first param must not match"))
	}
}

func TestBuildHTTPInterceptMiddleware_CallsNextAndRunsIntercept(t *testing.T) {
	c := newInterceptorTestContext()
	called := false
	c.Next = func() { called = true }

	var gotAgg *aggregation.Aggregation
	mw := BuildHTTPInterceptMiddleware("test.key", func(_ *ctx.HTTPContext, agg *aggregation.Aggregation) any {
		gotAgg = agg
		return "intercepted"
	})
	mw(c)

	if !called {
		t.Error(test.DiffMessage(called, true, "BuildHTTPInterceptMiddleware must call Next"))
	}
	if gotAgg == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil aggregation", "the intercept fn must receive an aggregation"))
	}
	if gotAgg.InterceptorData != "intercepted" {
		t.Error(test.DiffMessage(gotAgg.InterceptorData, "intercepted", "the intercept fn's return value must be stored as InterceptorData"))
	}
	if gotAgg.IsMainHandlerCalled {
		t.Error(test.DiffMessage(gotAgg.IsMainHandlerCalled, false, "IsMainHandlerCalled must default to false before the chain runs Pipe"))
	}
}

func TestBuildHTTPInterceptMiddleware_StacksAggregationsUnderSameKey(t *testing.T) {
	c := newInterceptorTestContext()
	c.Next = func() {}

	mw := BuildHTTPInterceptMiddleware("test.key", func(_ *ctx.HTTPContext, agg *aggregation.Aggregation) any {
		return nil
	})
	mw(c)
	mw(c)

	aggregations, ok := c.Request.Context().Value(WithValueKey("test.key")).([]*aggregation.Aggregation)
	if !ok {
		t.Fatal(test.DiffMessage(nil, "[]*aggregation.Aggregation", "aggregations must be stored under WithValueKey(key)"))
	}
	if len(aggregations) != 2 {
		t.Error(test.DiffMessage(len(aggregations), 2, "a second interceptor on the same key must append, not overwrite"))
	}
}
