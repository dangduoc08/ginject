package core

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/trace"
)

func returnHTTPOld(c *ctx.HTTPContext, data reflect.Value) {
	switch data.Type().Kind() {
	case
		reflect.Map,
		reflect.Slice,
		reflect.Struct,
		reflect.Interface:
		c.JSON(data.Interface())
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		c.Text(fmt.Sprint(data))
	case
		reflect.Pointer,
		reflect.UnsafePointer:
		c.Text(fmt.Sprint(data.UnsafePointer()))
	case
		reflect.String:
		c.Text(data.Interface().(string))
	case
		reflect.Func:
		c.Text(data.Type().String())
	}
}

func toWSMessageOld(data reflect.Value) string {
	switch data.Type().Kind() {
	case
		reflect.Map,
		reflect.Slice,
		reflect.Struct,
		reflect.Interface:
		return fmt.Sprint(data.Interface())
	case
		reflect.Bool,
		reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64,
		reflect.Complex64,
		reflect.Complex128:
		return fmt.Sprint(data)
	case
		reflect.Pointer,
		reflect.UnsafePointer:
		return fmt.Sprint(data.UnsafePointer())
	case
		reflect.String:
		return data.Interface().(string)
	case
		reflect.Func:
		return data.Type().String()
	default:
		return data.String()
	}
}

func traceHTTPHandlerOld(ev *event.Event, stage, name string, h ctx.HTTPHandler) ctx.HTTPHandler {
	return func(c *ctx.HTTPContext) {
		start := time.Now()
		defer func() {
			ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: stage, Name: name, Duration: time.Since(start)})
		}()
		h(c)
	}
}

func traceHTTPCatchOld(ev *event.Event, name string, fn common.HTTPCatch) common.HTTPCatch {
	return func(c *ctx.HTTPContext, ex *exception.Exception) {
		start := time.Now()
		defer func() {
			ev.Emit(trace.EventName, trace.Event{ID: c.GetID(), Stage: trace.StageExceptionFilter, Name: name, Duration: time.Since(start)})
		}()
		fn(c, ex)
	}
}

func BenchmarkGetFnArgsByType_NonPipeable(b *testing.B) {
	handler := func(*ctx.HTTPContext, *http.Request, http.ResponseWriter) {}
	fType := reflect.TypeOf(handler)
	injectedProviders := map[string]Provider{}
	cb := func(string, int, reflect.Value) {}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getFnArgsByType(fType, injectedProviders, cb)
	}
}

func BenchmarkGetFnArgsByType_MixedPipeable(b *testing.B) {
	handler := func(*ctx.HTTPContext, fnContextPipeableDTO, fnQueryPipeableDTO) {}
	fType := reflect.TypeOf(handler)
	injectedProviders := map[string]Provider{
		genFieldKey(reflect.TypeOf(fnTestProvider{})): fnTestProvider{Tag: "injected"},
	}
	cb := func(string, int, reflect.Value) {}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getFnArgsByType(fType, injectedProviders, cb)
	}
}

func BenchmarkGetPkgFromControllerKey(b *testing.B) {
	key := "[123456789]github.com/dangduoc08/ginject/sample/modules/user.UserController"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		getPkgFromControllerKey(key)
	}
}

func BenchmarkGenFieldKey(b *testing.B) {
	t := reflect.TypeOf(&mockProvider{})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		genFieldKey(t)
	}
}

func BenchmarkGenControllerKey(b *testing.B) {
	m := ModuleBuilder().Build()
	c := &mockController{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		genControllerKey(m, c)
	}
}

func BenchmarkIsDynamicModule(b *testing.B) {
	s := "func(pkg.Provider) *core.Module"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		isDynamicModule(s) //nolint:errcheck
	}
}

func BenchmarkToUniqueControllers(b *testing.B) {
	m := ModuleBuilder().Build()
	c1 := &mockController{}
	c2 := &mockController{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controllers := []Controller{c1, c2, c1, c2, c1}
		toUniqueControllers(m, &controllers)
	}
}

func BenchmarkReturnHTTPString_Old(b *testing.B) {
	c := newHTTPContext()
	v := reflect.ValueOf("hello world")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		returnHTTPOld(c, v)
	}
}

func BenchmarkReturnHTTPString_New(b *testing.B) {
	c := newHTTPContext()
	v := reflect.ValueOf("hello world")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		returnHTTP(c, v)
	}
}

func BenchmarkReturnHTTPInt_Old(b *testing.B) {
	c := newHTTPContext()
	v := reflect.ValueOf(123456)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		returnHTTPOld(c, v)
	}
}

func BenchmarkReturnHTTPInt_New(b *testing.B) {
	c := newHTTPContext()
	v := reflect.ValueOf(123456)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		returnHTTP(c, v)
	}
}

func BenchmarkReturnHTTPFloat_Old(b *testing.B) {
	c := newHTTPContext()
	v := reflect.ValueOf(3.14159)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		returnHTTPOld(c, v)
	}
}

func BenchmarkReturnHTTPFloat_New(b *testing.B) {
	c := newHTTPContext()
	v := reflect.ValueOf(3.14159)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		returnHTTP(c, v)
	}
}

func BenchmarkToWSMessageString_Old(b *testing.B) {
	v := reflect.ValueOf("hello world")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		toWSMessageOld(v)
	}
}

func BenchmarkToWSMessageString_New(b *testing.B) {
	v := reflect.ValueOf("hello world")
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		toWSMessage(v)
	}
}

func BenchmarkToWSMessageInt_Old(b *testing.B) {
	v := reflect.ValueOf(123456)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		toWSMessageOld(v)
	}
}

func BenchmarkToWSMessageInt_New(b *testing.B) {
	v := reflect.ValueOf(123456)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		toWSMessage(v)
	}
}

func BenchmarkTraceHTTPHandler_NoListener_Old(b *testing.B) {
	ev := event.NewEvent()
	c := newHTTPContext()
	h := traceHTTPHandlerOld(ev, trace.StageGuard, "bench.Guard", func(c *ctx.HTTPContext) {})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		h(c)
	}
}

func BenchmarkTraceHTTPHandler_NoListener_New(b *testing.B) {
	ev := event.NewEvent()
	c := newHTTPContext()
	h := traceHTTPHandler(ev, trace.StageGuard, "bench.Guard", func(c *ctx.HTTPContext) {})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		h(c)
	}
}

func BenchmarkTraceHTTPHandler_WithListener_Old(b *testing.B) {
	ev := event.NewEvent()
	ev.On(trace.EventName, func(args ...any) {})
	c := newHTTPContext()
	h := traceHTTPHandlerOld(ev, trace.StageGuard, "bench.Guard", func(c *ctx.HTTPContext) {})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		h(c)
	}
}

func BenchmarkTraceHTTPHandler_WithListener_New(b *testing.B) {
	ev := event.NewEvent()
	ev.On(trace.EventName, func(args ...any) {})
	c := newHTTPContext()
	h := traceHTTPHandler(ev, trace.StageGuard, "bench.Guard", func(c *ctx.HTTPContext) {})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		h(c)
	}
}

func BenchmarkTraceHTTPCatch_NoListener_Old(b *testing.B) {
	ev := event.NewEvent()
	c := newHTTPContext()
	ex := exception.BadRequestException("bad")
	fn := traceHTTPCatchOld(ev, "bench.Filter", func(c *ctx.HTTPContext, ex *exception.Exception) {})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		fn(c, &ex)
	}
}

func BenchmarkTraceHTTPCatch_NoListener_New(b *testing.B) {
	ev := event.NewEvent()
	c := newHTTPContext()
	ex := exception.BadRequestException("bad")
	fn := traceHTTPCatch(ev, "bench.Filter", func(c *ctx.HTTPContext, ex *exception.Exception) {})
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		fn(c, &ex)
	}
}

func BenchmarkBuildUseMiddleware_NoListener(b *testing.B) {
	ev := event.NewEvent()
	c := newHTTPContext()
	c.Next = func() {}
	mw := buildUseMiddleware(func(r *http.Request, w http.ResponseWriter, next ctx.Next) { next() }, ev, "bench.Middleware", trace.TransportHTTP)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mw(c)
	}
}

func BenchmarkBuildUseMiddleware_WithListener(b *testing.B) {
	ev := event.NewEvent()
	ev.On(trace.EventName, func(args ...any) {})
	c := newHTTPContext()
	c.Next = func() {}
	mw := buildUseMiddleware(func(r *http.Request, w http.ResponseWriter, next ctx.Next) { next() }, ev, "bench.Middleware", trace.TransportHTTP)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		mw(c)
	}
}
