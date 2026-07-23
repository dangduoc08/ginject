package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
)

func TestInjectProvidersIntoHTTPExceptionFilters_Empty(t *testing.T) {
	e := &ExceptionFilter{}
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	items := e.InjectProvidersIntoHTTPExceptionFilters(r, noopCB)
	if len(items) != 0 {
		t.Error(test.DiffMessage(len(items), 0, "no bound filters → empty result"))
	}
}

func TestInjectProvidersIntoHTTPExceptionFilters_ApplyAll(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(mockExFilter{})

	r := buildHTTP(map[string]string{
		"READ_users":    "/users/",
		"CREATE_orders": "/orders/",
	})

	items := e.InjectProvidersIntoHTTPExceptionFilters(r, noopCB)
	if len(items) != 2 {
		t.Error(test.DiffMessage(len(items), 2, "filter with no handlers applies to all patterns"))
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

func TestInjectProvidersIntoHTTPExceptionFilters_MainHandlerName(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(mockExFilter{})

	r := buildHTTP(map[string]string{"READ_items": "/items/"})
	items := e.InjectProvidersIntoHTTPExceptionFilters(r, noopCB)

	if len(items) != 1 {
		t.Error(test.DiffMessage(len(items), 1, "one pattern → one item"))
		return
	}
	if items[0].HTTP.Common.MainHandlerName != "READ_items" {
		t.Error(test.DiffMessage(items[0].HTTP.Common.MainHandlerName, "READ_items", "main handler name"))
	}
}

func TestInjectProvidersIntoHTTPExceptionFilters_HandlerIsCallableCatch(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(mockExFilter{})

	r := buildHTTP(map[string]string{"READ_users": "/users/"})
	items := e.InjectProvidersIntoHTTPExceptionFilters(r, noopCB)
	if len(items) != 1 {
		t.Fatal(test.DiffMessage(len(items), 1, "one pattern → one item"))
	}

	if _, ok := items[0].HTTP.Common.Handler.(HTTPCatch); !ok {
		t.Fatal(test.DiffMessage(items[0].HTTP.Common.Handler, "HTTPCatch", "Handler must be callable as HTTPCatch"))
	}
}

func TestInjectProvidersIntoHTTPExceptionFilters_NoCatch_Panics(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(noCatchExFilter{})
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "filter with no Catch method must panic"))
		}
	}()
	e.InjectProvidersIntoHTTPExceptionFilters(r, noopCB)
}

func TestInjectProvidersIntoHTTPExceptionFilters_WrongParamType_Panics(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(wrongParamExFilter{})
	r := buildHTTP(map[string]string{"READ_users": "/users/"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "Catch with a non-context first param must panic"))
		}
	}()
	e.InjectProvidersIntoHTTPExceptionFilters(r, noopCB)
}

func TestAsHTTPExceptionFilter_Valid(t *testing.T) {
	fn, ok := AsHTTPExceptionFilter(mockExFilter{})
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "mockExFilter must match HTTPCatch"))
	}
	ex := exception.InternalServerErrorException("")
	fn(nil, &ex)
}

func TestAsHTTPExceptionFilter_NoMethod(t *testing.T) {
	_, ok := AsHTTPExceptionFilter(noCatchExFilter{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a filter with no Catch must not match"))
	}
}

func TestAsHTTPExceptionFilter_WrongShape(t *testing.T) {
	_, ok := AsHTTPExceptionFilter(wrongParamExFilter{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a Catch with a non-context first param must not match"))
	}
}

func TestRunHTTPCatchChain_InvokesCatch(t *testing.T) {
	c := ctx.NewHTTPContext()

	var gotEx *exception.Exception
	RunHTTPCatchChain(c, []HTTPCatch{
		func(_ *ctx.HTTPContext, ex *exception.Exception) { gotEx = ex },
	}, "boom")

	if gotEx == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil exception", "RunHTTPCatchChain must invoke the catch function"))
	}
	if gotEx.GetMessage() != "boom" {
		t.Error(test.DiffMessage(gotEx.GetMessage(), "boom", "the recovered value must be normalized before being passed to Catch"))
	}
}

func TestRunHTTPCatchChain_FallsBackToNextOnPanic(t *testing.T) {
	c := ctx.NewHTTPContext()

	secondCalled := false
	RunHTTPCatchChain(c, []HTTPCatch{
		func(*ctx.HTTPContext, *exception.Exception) { panic("filter itself panics") },
		func(*ctx.HTTPContext, *exception.Exception) { secondCalled = true },
	}, "boom")

	if !secondCalled {
		t.Error(test.DiffMessage(secondCalled, true, "a panicking catch fn must fall back to the next one in the chain"))
	}
}

func TestRunHTTPCatchChain_EmptyIsNoop(t *testing.T) {
	c := ctx.NewHTTPContext()
	RunHTTPCatchChain(c, nil, "boom")
}
