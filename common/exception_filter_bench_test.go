package common

import (
	"errors"
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

func BenchmarkNormalizeRecovered(b *testing.B) {
	err := errors.New("boom")
	b.ResetTimer()
	for range b.N {
		_ = NormalizeRecovered(err)
	}
}
