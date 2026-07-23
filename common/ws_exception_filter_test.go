package common

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
)

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
	e.BindExceptionFilter(mockWSExFilter{})

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

func TestInjectProvidersIntoWSExceptionFilters_HandlerIsCallableCatch(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(mockWSExFilter{})

	ws := buildWS(map[string]string{"message": "ON_message"})
	items := e.InjectProvidersIntoWSExceptionFilters(ws, noopCB)
	if len(items) != 1 {
		t.Fatal(test.DiffMessage(len(items), 1, "one pattern → one item"))
	}

	if _, ok := items[0].WS.Common.Handler.(WSCatch); !ok {
		t.Fatal(test.DiffMessage(items[0].WS.Common.Handler, "WSCatch", "Handler must be callable as WSCatch"))
	}
}

func TestInjectProvidersIntoWSExceptionFilters_NoCatch_Panics(t *testing.T) {
	e := &ExceptionFilter{}
	e.BindExceptionFilter(noCatchExFilter{})
	ws := buildWS(map[string]string{"message": "ON_message"})

	defer func() {
		if rec := recover(); rec == nil {
			t.Error(test.DiffMessage(nil, "panic", "filter with no Catch method must panic"))
		}
	}()
	e.InjectProvidersIntoWSExceptionFilters(ws, noopCB)
}

func TestAsWSExceptionFilter_Valid(t *testing.T) {
	fn, ok := AsWSExceptionFilter(mockWSExFilter{})
	if !ok {
		t.Fatal(test.DiffMessage(ok, true, "mockWSExFilter must match WSCatch"))
	}
	ex := exception.InternalServerErrorException("")
	fn(nil, &ex)
}

func TestAsWSExceptionFilter_NoMethod(t *testing.T) {
	_, ok := AsWSExceptionFilter(noCatchExFilter{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a filter with no Catch must not match"))
	}
}

func TestAsWSExceptionFilter_WrongShape(t *testing.T) {
	_, ok := AsWSExceptionFilter(wrongParamExFilter{})
	if ok {
		t.Error(test.DiffMessage(true, false, "a Catch with a non-context first param must not match"))
	}
}

func TestRunWSCatchChain_InvokesCatch(t *testing.T) {
	c := ctx.NewWSContext()

	var gotEx *exception.Exception
	RunWSCatchChain(c, []WSCatch{
		func(_ *ctx.WSContext, ex *exception.Exception) { gotEx = ex },
	}, "boom")

	if gotEx == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil exception", "RunWSCatchChain must invoke the catch function"))
	}
	if gotEx.GetMessage() != "boom" {
		t.Error(test.DiffMessage(gotEx.GetMessage(), "boom", "the recovered value must be normalized before being passed to Catch"))
	}
}

func TestRunWSCatchChain_FallsBackToNextOnPanic(t *testing.T) {
	c := ctx.NewWSContext()

	secondCalled := false
	RunWSCatchChain(c, []WSCatch{
		func(*ctx.WSContext, *exception.Exception) { panic("filter itself panics") },
		func(*ctx.WSContext, *exception.Exception) { secondCalled = true },
	}, "boom")

	if !secondCalled {
		t.Error(test.DiffMessage(secondCalled, true, "a panicking catch fn must fall back to the next one in the chain"))
	}
}

func TestRunWSCatchChain_EmptyIsNoop(t *testing.T) {
	c := ctx.NewWSContext()
	RunWSCatchChain(c, nil, "boom")
}
