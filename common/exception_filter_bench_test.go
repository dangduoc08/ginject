package common

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/routing"
)

func buildBenchREST(n int) *REST {
	r := &REST{
		PatternToFuncNameMap: make(map[string]string, n),
		FuncNameToPatternMap: make(map[string]string, n),
	}
	for i := range n {
		fn := fmt.Sprintf("READ_resource%d", i)
		route := fmt.Sprintf("/resource%d/", i)
		p := routing.MethodRouteVersionToPattern("GET", route, "")
		r.PatternToFuncNameMap[p] = fn
		r.FuncNameToPatternMap[fn] = p
	}
	return r
}

func buildBenchWS(n int) *WS {
	ws := &WS{
		funcNameByEvent: make(map[string]string, n),
	}
	for i := range n {
		event := fmt.Sprintf("event%d", i)
		fn := fmt.Sprintf("ON_event%d", i)
		ws.funcNameByEvent[event] = fn
	}
	return ws
}

var benchCB = func(_ int, _ reflect.Type, _ reflect.Value, _ reflect.Value) {}

func BenchmarkInjectProvidersIntoRESTExceptionFilters_ApplyAll(b *testing.B) {
	r := buildBenchREST(20)
	b.ResetTimer()
	for range b.N {
		e := &ExceptionFilter{}
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.InjectProvidersIntoRESTExceptionFilters(r, benchCB)
	}
}

func BenchmarkInjectProvidersIntoWSExceptionFilters_ApplyAll(b *testing.B) {
	ws := buildBenchWS(20)
	b.ResetTimer()
	for range b.N {
		e := &ExceptionFilter{}
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.BindExceptionFilter(mockExFilter{})
		e.InjectProvidersIntoWSExceptionFilters(ws, benchCB)
	}
}
