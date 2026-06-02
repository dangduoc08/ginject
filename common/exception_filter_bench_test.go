package common

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/routing"
)

func buildBenchREST(n int) *REST {
	r := &REST{
		PatternToFnNameMap: make(map[string]string, n),
		FnNameToPatternMap: make(map[string]string, n),
	}
	for i := range n {
		fn := fmt.Sprintf("READ_resource%d", i)
		route := fmt.Sprintf("/resource%d/", i)
		p := routing.MethodRouteVersionToPattern("GET", route, "")
		r.PatternToFnNameMap[p] = fn
		r.FnNameToPatternMap[fn] = p
	}
	return r
}

func buildBenchWS(n int) *WS {
	ws := &WS{
		subprotocol:        "bench",
		patternToFnNameMap: make(map[string]string, n),
	}
	for i := range n {
		event := fmt.Sprintf("bench_/event%d/", i)
		fn := fmt.Sprintf("ON_event%d", i)
		ws.patternToFnNameMap[event] = fn
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
