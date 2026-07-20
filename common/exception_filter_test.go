package common

import (
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/routing"
)

type mockExFilter struct{}

func (mockExFilter) Catch(_ *ctx.HTTPContext, _ *exception.Exception) {}

type mockWSExFilter struct{}

func (mockWSExFilter) Catch(_ *ctx.WSContext, _ *exception.Exception) {}

type noCatchExFilter struct{}

type wrongParamExFilter struct{}

func (wrongParamExFilter) Catch(_ int, _ *exception.Exception) {}

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

func TestExceptionFilterShapeError_MessageContainsType(t *testing.T) {
	err := ExceptionFilterShapeError(noCatchExFilter{})
	if err == nil {
		t.Fatal(test.DiffMessage(nil, "non-nil error", "ExceptionFilterShapeError must not return nil"))
	}
	if !strings.Contains(err.Error(), "noCatchExFilter") {
		t.Error(test.DiffMessage(err.Error(), "contains noCatchExFilter", "error message should name the offending type"))
	}
}

func TestNormalizeRecovered_ExceptionPassthrough(t *testing.T) {
	original := exception.ForbiddenException("nope")
	got := NormalizeRecovered(original)
	if got.Error() != original.Error() {
		t.Error(test.DiffMessage(got.Error(), original.Error(), "an *exception.Exception panic must pass through unchanged"))
	}
}

func TestNormalizeRecovered_ErrorValue(t *testing.T) {
	got := NormalizeRecovered(errors.New("boom"))
	if got.GetResponse() != "boom" {
		t.Error(test.DiffMessage(got.GetResponse(), "boom", "an error panic should use its Error() text as the response"))
	}
}

func TestNormalizeRecovered_StringValue(t *testing.T) {
	got := NormalizeRecovered("boom string")
	if got.GetResponse() != "boom string" {
		t.Error(test.DiffMessage(got.GetResponse(), "boom string", "a string panic should be used verbatim as the response"))
	}
}

func TestNormalizeRecovered_UnknownTypeUsesGenericMessage(t *testing.T) {
	got := NormalizeRecovered(42)
	want := http.StatusText(http.StatusInternalServerError)
	if got.GetResponse() != want {
		t.Error(test.DiffMessage(got.GetResponse(), want, "an unrecognized panic value should fall back to the generic 500 text"))
	}
}
