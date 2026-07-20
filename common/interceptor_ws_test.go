package common

import (
	"testing"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
)

func newInterceptorTestWSContext() *ctx.WSContext {
	return ctx.NewWSContext()
}

func TestInjectProvidersIntoWSInterceptors_Empty(t *testing.T) {
	ic := &Interceptor{}
	ws := buildWS(map[string]string{"message": "ON_message"})

	items := ic.InjectProvidersIntoWSInterceptors(ws, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound interceptors → empty result"))
	}
}

func TestInjectProvidersIntoWSInterceptors_ApplyAll(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockWSInterceptable{})

	ws := buildWS(map[string]string{
		"message": "ON_message",
		"status":  "ON_status",
	})

	items := ic.InjectProvidersIntoWSInterceptors(ws, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "interceptor with no handlers applies to all WS patterns"))
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

func TestInjectProvidersIntoWSInterceptors_HandlerIsCallableIntercept(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(mockWSInterceptable{})

	ws := buildWS(map[string]string{"message": "ON_message"})
	items := ic.InjectProvidersIntoWSInterceptors(ws, noopCB)
	if len(items) != 1 {
		t.Fatal(test.DiffMessage(len(items), 1, "one pattern → one item"))
	}

	if _, ok := items[0].WS.Common.Handler.(WSIntercept); !ok {
		t.Fatal(test.DiffMessage(items[0].WS.Common.Handler, "WSIntercept", "Handler must be callable as WSIntercept"))
	}
}

func TestInjectProvidersIntoWSInterceptors_NoIntercept_Panics(t *testing.T) {
	ic := &Interceptor{}
	ic.BindInterceptor(noInterceptMethod{})
	ws := buildWS(map[string]string{"message": "ON_message"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "interceptor with no Intercept method must panic"))
		}
	}()
	ic.InjectProvidersIntoWSInterceptors(ws, noopCB)
}

func TestAsWSInterceptor_Valid(t *testing.T) {
	fn, ok := AsWSInterceptor(mockWSInterceptable{})
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "mockWSInterceptable must match WSIntercept"))
	}
	fn(nil, aggregation.NewAggregation())
}

func TestAsWSInterceptor_NoMethod(t *testing.T) {
	_, ok := AsWSInterceptor(noInterceptMethod{})
	if ok {
		t.Error(test.DiffMessage(true, false, "an interceptable with no Intercept must not match"))
	}
}

func TestAsWSInterceptor_WrongShape(t *testing.T) {
	_, ok := AsWSInterceptor(wrongParamInterceptable{})
	if ok {
		t.Error(test.DiffMessage(true, false, "an Intercept with a non-context first param must not match"))
	}
}

func TestBuildWSInterceptMiddleware_CallsNextAndRunsIntercept(t *testing.T) {
	c := newInterceptorTestWSContext()
	called := false
	c.Next = func() { called = true }

	var gotAgg *aggregation.Aggregation
	mw := BuildWSInterceptMiddleware("test.event", func(_ *ctx.WSContext, agg *aggregation.Aggregation) any {
		gotAgg = agg
		return "intercepted"
	})
	mw(c)

	if !called {
		t.Error(test.DiffMessage(called, true, "BuildWSInterceptMiddleware must call Next"))
	}
	if gotAgg == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil aggregation", "the intercept fn must receive an aggregation"))
	}
	if gotAgg.InterceptorData != "intercepted" {
		t.Error(test.DiffMessage(gotAgg.InterceptorData, "intercepted", "the intercept fn's return value must be stored as InterceptorData"))
	}
}
