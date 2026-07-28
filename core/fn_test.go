package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/event"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
	"github.com/dangduoc08/ginject/trace"
)

type mockProvider struct{}

func (m *mockProvider) NewProvider() Provider { return m }

type mockController struct{}

func (m *mockController) NewController() Controller { return m }

func TestGetPkgFromControllerKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"[123]github.com/foo/bar.Controller", "github.com/foo/bar.Controller"},
		{"[999]pkg.Type", "pkg.Type"},
		{"noBrackets", "noBrackets"},
		{"[a][b]pkg.Type", "pkg.Type"},
	}
	for _, c := range cases {
		got := getPkgFromControllerKey(c.in)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "getPkgFromControllerKey("+c.in+")"))
		}
	}
}

func TestGenFieldKey(t *testing.T) {
	t1 := reflect.TypeOf(mockProvider{})
	got := genFieldKey(t1)
	want := t1.PkgPath() + "/" + t1.String()
	if got != want {
		t.Error(test.DiffMessage(got, want, "genFieldKey"))
	}
}

func TestGenProviderKey(t *testing.T) {
	p := &mockProvider{}
	got := genProviderKey(p)
	want := genFieldKey(reflect.TypeOf(p))
	if got != want {
		t.Error(test.DiffMessage(got, want, "genProviderKey"))
	}
}

func TestGenControllerKey(t *testing.T) {
	m := ModuleBuilder().Build()
	c := &mockController{}
	got := genControllerKey(m, c)
	want := "[" + m.ID() + "]" + genFieldKey(reflect.TypeOf(c))
	if got != want {
		t.Error(test.DiffMessage(got, want, "genControllerKey"))
	}
}

func TestIsDynamicModule(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"func(pkg.Provider) *core.Module", true},
		{"func(pkg.A, pkg.B) *core.Module", true},
		{"func() *core.Module", true},
		{"*core.Module", false},
		{"func() error", false},
		{"", false},
	}
	for _, c := range cases {
		got := isDynamicModule(c.in)
		if got != c.want {
			t.Error(test.DiffMessage(got, c.want, "isDynamicModule("+c.in+")"))
		}
	}
}

func TestToUniqueControllers(t *testing.T) {
	m := ModuleBuilder().Build()
	c1 := &mockController{}
	c2 := &mockController{}

	controllers := []Controller{c1, c1, c2, c1}
	toUniqueControllers(m, &controllers)

	if len(controllers) != 1 {
		t.Error(test.DiffMessage(len(controllers), 1, "toUniqueControllers dedup"))
	}
}

func TestToUniqueControllersEmpty(t *testing.T) {
	m := ModuleBuilder().Build()
	controllers := []Controller{}
	toUniqueControllers(m, &controllers)
	if len(controllers) != 0 {
		t.Error(test.DiffMessage(len(controllers), 0, "toUniqueControllers empty"))
	}
}

func TestIsInjectedProvider(t *testing.T) {
	providerType := reflect.TypeOf(mockProvider{})
	notProviderType := reflect.TypeOf(struct{}{})

	if !isInjectedProvider(providerType) {
		t.Error(test.DiffMessage(false, true, "mockProvider should be injectable"))
	}
	if isInjectedProvider(notProviderType) {
		t.Error(test.DiffMessage(true, false, "anonymous struct should not be injectable"))
	}
}

func TestSetStatusCodeInt(t *testing.T) {
	c := newHTTPContext()
	setStatusCode(c, reflect.ValueOf(http.StatusOK))
	if c.Code != http.StatusOK {
		t.Error(test.DiffMessage(c.Code, http.StatusOK, "setStatusCode reflect.Int"))
	}
}

func TestSetStatusCodeInvalidInt(t *testing.T) {
	c := newHTTPContext()
	c.Status(http.StatusTeapot)
	setStatusCode(c, reflect.ValueOf(9999))
	if c.Code != http.StatusTeapot {
		t.Error(test.DiffMessage(c.Code, http.StatusTeapot, "setStatusCode invalid int should not change status"))
	}
}

func TestSetStatusCodeInterfaceValid(t *testing.T) {
	c := newHTTPContext()
	var v any = http.StatusCreated
	setStatusCode(c, reflect.ValueOf(v))
	if c.Code != http.StatusCreated {
		t.Error(test.DiffMessage(c.Code, http.StatusCreated, "setStatusCode interface valid int"))
	}
}

func TestSetStatusCodeInterfaceInvalid(t *testing.T) {
	c := newHTTPContext()
	c.Status(http.StatusTeapot)
	var v any = "not-an-int"
	setStatusCode(c, reflect.ValueOf(v))
	if c.Code != http.StatusTeapot {
		t.Error(test.DiffMessage(c.Code, http.StatusTeapot, "setStatusCode interface non-int should not change status"))
	}
}

func TestReturnHTTPString(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnHTTP(c, reflect.ValueOf("hello"))
	if w.Body.String() != "hello" {
		t.Error(test.DiffMessage(w.Body.String(), "hello", "returnHTTP string"))
	}
}

func TestReturnHTTPMap(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnHTTP(c, reflect.ValueOf(map[string]any{"k": "v"}))
	if w.Body.Len() == 0 {
		t.Error(test.DiffMessage(0, ">0", "returnHTTP map should produce JSON body"))
	}
}

func TestReturnHTTPInt(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnHTTP(c, reflect.ValueOf(42))
	if w.Body.String() != "42" {
		t.Error(test.DiffMessage(w.Body.String(), "42", "returnHTTP int"))
	}
}

func TestReturnHTTPBool(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnHTTP(c, reflect.ValueOf(true))
	if w.Body.String() != "true" {
		t.Error(test.DiffMessage(w.Body.String(), "true", "returnHTTP bool"))
	}
}

func TestReturnHTTPSlice(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnHTTP(c, reflect.ValueOf([]int{1, 2, 3}))
	if w.Body.Len() == 0 {
		t.Error(test.DiffMessage(0, ">0", "returnHTTP slice should produce JSON body"))
	}
}

func TestToWSMessageString(t *testing.T) {
	got := toWSMessage(reflect.ValueOf("hello"))
	if got != "hello" {
		t.Error(test.DiffMessage(got, "hello", "toWSMessage string"))
	}
}

func TestToWSMessageInt(t *testing.T) {
	got := toWSMessage(reflect.ValueOf(42))
	if got != "42" {
		t.Error(test.DiffMessage(got, "42", "toWSMessage int"))
	}
}

func TestToWSMessageBool(t *testing.T) {
	got := toWSMessage(reflect.ValueOf(true))
	if got != "true" {
		t.Error(test.DiffMessage(got, "true", "toWSMessage bool"))
	}
}

func TestToWSMessageMap(t *testing.T) {
	got := toWSMessage(reflect.ValueOf(map[string]any{"k": "v"}))
	if got == "" {
		t.Error(test.DiffMessage(got, "non-empty JSON", "toWSMessage map"))
	}
}

func TestToWSMessageSlice(t *testing.T) {
	got := toWSMessage(reflect.ValueOf([]int{1, 2}))
	if got == "" {
		t.Error(test.DiffMessage(got, "non-empty JSON", "toWSMessage slice"))
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := getLocalIP()
	if ip != "" {
		if net := reflect.TypeOf(ip).Kind(); net != reflect.String {
			t.Error(test.DiffMessage(net, reflect.String, "getLocalIP should return string"))
		}
	}
}

func TestGetDependencyContext(t *testing.T) {
	c := newHTTPContext()
	got := getHTTPDependency(httpContextKey, c, reflect.Value{})
	if got != c {
		t.Error(test.DiffMessage(got, c, "getHTTPDependency httpContextKey"))
	}
}

func TestGetDependencyRequest(t *testing.T) {
	c := newHTTPContext()
	got := getHTTPDependency(requestKey, c, reflect.Value{})
	if got != c.Request {
		t.Error(test.DiffMessage(got, c.Request, "getHTTPDependency requestKey"))
	}
}

func TestGetDependencyResponse(t *testing.T) {
	c := newHTTPContext()
	got := getHTTPDependency(responseKey, c, reflect.Value{})
	if got != c.ResponseWriter {
		t.Error(test.DiffMessage(got, c.ResponseWriter, "getHTTPDependency responseKey"))
	}
}

func TestGetDependencyUnknownReturnsDependencies(t *testing.T) {
	c := newHTTPContext()
	got := getHTTPDependency("unknown-key", c, reflect.Value{})
	if got == nil {
		t.Error(test.DiffMessage(nil, "dependencies map", "getHTTPDependency unknown key should return dependencies"))
	}
}

type fnTestProvider struct{ Tag string }

func (p fnTestProvider) NewProvider() Provider { return p }

type fnContextPipeableDTO struct{ P fnTestProvider }

func (d fnContextPipeableDTO) Transform(*ctx.HTTPContext, common.ArgumentMetadata) any { return nil }

type fnBodyPipeableDTO struct{ P fnTestProvider }

func (d fnBodyPipeableDTO) Transform(ctx.Body, common.ArgumentMetadata) any { return nil }

type fnFormPipeableDTO struct{}

func (d fnFormPipeableDTO) Transform(ctx.Form, common.ArgumentMetadata) any { return nil }

type fnQueryPipeableDTO struct{}

func (d fnQueryPipeableDTO) Transform(ctx.Query, common.ArgumentMetadata) any { return nil }

type fnHeaderPipeableDTO struct{}

func (d fnHeaderPipeableDTO) Transform(ctx.Header, common.ArgumentMetadata) any { return nil }

type fnParamPipeableDTO struct{}

func (d fnParamPipeableDTO) Transform(ctx.Param, common.ArgumentMetadata) any { return nil }

type fnFilePipeableDTO struct{}

func (d fnFilePipeableDTO) Transform(ctx.File, common.ArgumentMetadata) any { return nil }

type fnWSPayloadPipeableDTO struct{}

func (d fnWSPayloadPipeableDTO) Transform(ctx.WSPayload, common.ArgumentMetadata) any { return nil }

func TestGetFnArgsByType_NonPipeableResolvesTypeKey(t *testing.T) {
	handler := func(*ctx.HTTPContext) {}
	fType := reflect.TypeOf(handler)

	var gotKey string
	var gotIndex int
	getFnArgsByType(fType, nil, func(key string, i int, _ reflect.Value) {
		gotKey = key
		gotIndex = i
	})

	if gotKey != httpContextKey {
		t.Error(test.DiffMessage(gotKey, httpContextKey, "non-pipeable param should resolve to its PkgPath+String key"))
	}
	if gotIndex != 0 {
		t.Error(test.DiffMessage(gotIndex, 0, "single param should be reported at index 0"))
	}
}

func TestGetFnArgsByType_ContextPipeableInjectsProvider(t *testing.T) {
	handler := func(fnContextPipeableDTO) {}
	fType := reflect.TypeOf(handler)
	injectedProviders := map[string]Provider{
		genFieldKey(reflect.TypeOf(fnTestProvider{})): fnTestProvider{Tag: "injected"},
	}

	var gotKey string
	var gotValue reflect.Value
	getFnArgsByType(fType, injectedProviders, func(key string, i int, v reflect.Value) {
		gotKey = key
		gotValue = v
	})

	if gotKey != common.ContextPipeableKey {
		t.Error(test.DiffMessage(gotKey, common.ContextPipeableKey, "ContextPipeable param should resolve to ContextPipeableKey"))
	}
	dto, ok := gotValue.Interface().(*fnContextPipeableDTO)
	if !ok {
		t.Fatalf("expected resolved value to be a *fnContextPipeableDTO, got %T", gotValue.Interface())
	}
	if dto.P.Tag != "injected" {
		t.Error(test.DiffMessage(dto.P.Tag, "injected", "ContextPipeable DTO's provider field should be injected from injectedProviders"))
	}
}

func TestGetFnArgsByType_BodyPipeableInjectsProvider(t *testing.T) {
	handler := func(fnBodyPipeableDTO) {}
	fType := reflect.TypeOf(handler)
	injectedProviders := map[string]Provider{
		genFieldKey(reflect.TypeOf(fnTestProvider{})): fnTestProvider{Tag: "injected"},
	}

	var gotKey string
	var gotValue reflect.Value
	getFnArgsByType(fType, injectedProviders, func(key string, i int, v reflect.Value) {
		gotKey = key
		gotValue = v
	})

	if gotKey != common.BodyPipeableKey {
		t.Error(test.DiffMessage(gotKey, common.BodyPipeableKey, "BodyPipeable param should resolve to BodyPipeableKey"))
	}
	dto := gotValue.Interface().(*fnBodyPipeableDTO)
	if dto.P.Tag != "injected" {
		t.Error(test.DiffMessage(dto.P.Tag, "injected", "BodyPipeable DTO's provider field should be injected from injectedProviders"))
	}
}

func TestGetFnArgsByType_AllPipeableKindsResolveTheirKey(t *testing.T) {
	cases := []struct {
		name    string
		handler any
		wantKey string
	}{
		{"form", func(fnFormPipeableDTO) {}, common.FormPipeableKey},
		{"query", func(fnQueryPipeableDTO) {}, common.QueryPipeableKey},
		{"header", func(fnHeaderPipeableDTO) {}, common.HeaderPipeableKey},
		{"param", func(fnParamPipeableDTO) {}, common.ParamPipeableKey},
		{"file", func(fnFilePipeableDTO) {}, common.FilePipeableKey},
		{"wsPayload", func(fnWSPayloadPipeableDTO) {}, common.WSPayloadPipeableKey},
	}

	for _, c := range cases {
		var gotKey string
		getFnArgsByType(reflect.TypeOf(c.handler), map[string]Provider{}, func(key string, i int, _ reflect.Value) {
			gotKey = key
		})
		if gotKey != c.wantKey {
			t.Error(test.DiffMessage(gotKey, c.wantKey, c.name+"Pipeable param should resolve to "+c.wantKey))
		}
	}
}

func TestGetFnArgsByType_MultipleParamsResolveInOrder(t *testing.T) {
	handler := func(*ctx.HTTPContext, fnQueryPipeableDTO, *http.Request) {}
	fType := reflect.TypeOf(handler)

	var gotKeys []string
	var gotIndexes []int
	getFnArgsByType(fType, map[string]Provider{}, func(key string, i int, _ reflect.Value) {
		gotKeys = append(gotKeys, key)
		gotIndexes = append(gotIndexes, i)
	})

	wantKeys := []string{httpContextKey, common.QueryPipeableKey, requestKey}
	for i, want := range wantKeys {
		if gotKeys[i] != want {
			t.Error(test.DiffMessage(gotKeys[i], want, "param order should be preserved"))
		}
		if gotIndexes[i] != i {
			t.Error(test.DiffMessage(gotIndexes[i], i, "param index should match its position"))
		}
	}
}

// classifyArgType memoizes its result in a shared sync.Map keyed by
// reflect.Type, so concurrent first-time classification of the same and
// different handler signatures must not race or produce inconsistent keys.
func TestGetFnArgsByType_ConcurrentCallsNoDataRace(t *testing.T) {
	handlers := []any{
		func(*ctx.HTTPContext) {},
		func(fnContextPipeableDTO) {},
		func(fnBodyPipeableDTO) {},
		func(*http.Request, http.ResponseWriter) {},
	}
	injectedProviders := map[string]Provider{
		genFieldKey(reflect.TypeOf(fnTestProvider{})): fnTestProvider{Tag: "injected"},
	}

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		for _, h := range handlers {
			wg.Add(1)
			go func(handler any) {
				defer wg.Done()
				getFnArgsByType(reflect.TypeOf(handler), injectedProviders, func(string, int, reflect.Value) {})
			}(h)
		}
	}
	wg.Wait()
}

type tracedQueryPipeDTO struct{}

func (d tracedQueryPipeDTO) Transform(ctx.Query, common.ArgumentMetadata) any {
	return tracedQueryPipeDTO{}
}

type tracedWSPayloadPipeDTO struct{}

func (d tracedWSPayloadPipeDTO) Transform(ctx.WSPayload, common.ArgumentMetadata) any {
	return tracedWSPayloadPipeDTO{}
}

func TestInvokeHTTPHandlerByProviders_PipeEmitsStagePipeEvent(t *testing.T) {
	ev := event.NewEvent()
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		if te.Stage == trace.StagePipe {
			d := te
			got = &d
		}
	})

	c := newHTTPContext()
	handler := func(tracedQueryPipeDTO) {}
	_, pipeElapsed := invokeHTTPHandlerByProviders(handler, nil, c, ev)

	if got == nil {
		t.Fatal("expected a pipe trace event for a Pipeable param")
	}
	if got.Stage != trace.StagePipe || got.Transport != trace.TransportHTTP {
		t.Error(test.DiffMessage([]any{got.Stage, got.Transport}, []any{trace.StagePipe, trace.TransportHTTP}, "pipe trace event stage/transport"))
	}
	if !strings.Contains(got.Name, "tracedQueryPipeDTO") {
		t.Error(test.DiffMessage(got.Name, "contains tracedQueryPipeDTO", "pipe trace event name should identify the concrete pipe type"))
	}
	if pipeElapsed <= 0 {
		t.Error(test.DiffMessage(pipeElapsed, ">0", "invokeHTTPHandlerByProviders should return the accumulated pipe duration"))
	}
}

func TestInvokeHTTPHandlerByProviders_NonPipeableParamDoesNotEmitStagePipe(t *testing.T) {
	ev := event.NewEvent()
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		got = &te
	})

	c := newHTTPContext()
	handler := func(*ctx.HTTPContext) {}
	_, pipeElapsed := invokeHTTPHandlerByProviders(handler, nil, c, ev)

	if got != nil {
		t.Error(test.DiffMessage(got.Stage, "<no event>", "a non-Pipeable param must not emit any trace event"))
	}
	if pipeElapsed != 0 {
		t.Error(test.DiffMessage(pipeElapsed, time.Duration(0), "a non-Pipeable param must not contribute to pipe duration"))
	}
}

func TestInvokeHTTPHandlerByProviders_NoListeners_SkipsPipeTracing(t *testing.T) {
	ev := event.NewEvent()
	c := newHTTPContext()
	handler := func(tracedQueryPipeDTO) {}
	_, pipeElapsed := invokeHTTPHandlerByProviders(handler, nil, c, ev)

	if pipeElapsed != 0 {
		t.Error(test.DiffMessage(pipeElapsed, time.Duration(0), "with no trace listeners attached, pipe timing should be skipped entirely"))
	}
}

func TestInvokeWSHandlerByProviders_PipeEmitsStagePipeEvent(t *testing.T) {
	ev := event.NewEvent()
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		if te.Stage == trace.StagePipe {
			d := te
			got = &d
		}
	})

	c := ctx.NewWSContext()
	handler := func(tracedWSPayloadPipeDTO) {}
	_, pipeElapsed := invokeWSHandlerByProviders(handler, nil, c, ev)

	if got == nil {
		t.Fatal("expected a pipe trace event for a WSPayloadPipeable param")
	}
	if got.Stage != trace.StagePipe || got.Transport != trace.TransportWS {
		t.Error(test.DiffMessage([]any{got.Stage, got.Transport}, []any{trace.StagePipe, trace.TransportWS}, "pipe trace event stage/transport"))
	}
	if pipeElapsed <= 0 {
		t.Error(test.DiffMessage(pipeElapsed, ">0", "invokeWSHandlerByProviders should return the accumulated pipe duration"))
	}
}

func TestInvokeWSHandlerByProviders_NonPipeableParamDoesNotEmitStagePipe(t *testing.T) {
	ev := event.NewEvent()
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		got = &te
	})

	c := ctx.NewWSContext()
	handler := func(*ctx.WSContext) {}
	_, pipeElapsed := invokeWSHandlerByProviders(handler, nil, c, ev)

	if got != nil {
		t.Error(test.DiffMessage(got.Stage, "<no event>", "a non-Pipeable param must not emit any trace event"))
	}
	if pipeElapsed != 0 {
		t.Error(test.DiffMessage(pipeElapsed, time.Duration(0), "a non-Pipeable param must not contribute to pipe duration"))
	}
}

func TestIsInjectableHandlerValid(t *testing.T) {
	handler := func(c *ctx.HTTPContext) {}
	err := isInjectableHandler(handler, nil, knownHTTPDependencyKeys)
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "isInjectableHandler valid handler"))
	}
}

func TestIsInjectableHandlerInvalid(t *testing.T) {
	type unknownType struct{}
	handler := func(_ unknownType) {}
	err := isInjectableHandler(handler, nil, knownHTTPDependencyKeys)
	if err == nil {
		t.Error(test.DiffMessage(nil, "error", "isInjectableHandler with unknown arg should return error"))
	}
}

func TestReturnHTTPFloat(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnHTTP(c, reflect.ValueOf(3.5))
	if w.Body.String() != "3.5" {
		t.Error(test.DiffMessage(w.Body.String(), "3.5", "returnHTTP float64"))
	}
}

func TestReturnHTTPUint(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnHTTP(c, reflect.ValueOf(uint(7)))
	if w.Body.String() != "7" {
		t.Error(test.DiffMessage(w.Body.String(), "7", "returnHTTP uint"))
	}
}

func TestToWSMessageFloat(t *testing.T) {
	got := toWSMessage(reflect.ValueOf(3.5))
	if got != "3.5" {
		t.Error(test.DiffMessage(got, "3.5", "toWSMessage float64"))
	}
}

func TestToWSMessageUint(t *testing.T) {
	got := toWSMessage(reflect.ValueOf(uint(7)))
	if got != "7" {
		t.Error(test.DiffMessage(got, "7", "toWSMessage uint"))
	}
}

func TestTraceHTTPHandler_NoListener_StillCallsWrapped(t *testing.T) {
	ev := event.NewEvent()
	c := newHTTPContext()
	var called bool
	h := traceHTTPHandler(ev, trace.StageGuard, "Test.Guard", func(c *ctx.HTTPContext) { called = true })
	h(c)
	if !called {
		t.Error(test.DiffMessage(false, true, "wrapped handler should still run when no listener is attached"))
	}
}

func TestTraceHTTPHandler_EmitsOnlyWhenListenerAttached(t *testing.T) {
	ev := event.NewEvent()
	c := newHTTPContext()
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		got = &te
	})
	h := traceHTTPHandler(ev, trace.StageGuard, "Test.Guard", func(c *ctx.HTTPContext) {})
	h(c)
	if got == nil {
		t.Fatal("expected a trace event to be emitted when a listener is attached")
	}
	if got.Stage != trace.StageGuard || got.Name != "Test.Guard" {
		t.Error(test.DiffMessage([]any{got.Stage, got.Name}, []any{trace.StageGuard, "Test.Guard"}, "traceHTTPHandler emitted event fields"))
	}
}

func TestTraceHTTPCatch_NoListener_StillCallsWrapped(t *testing.T) {
	ev := event.NewEvent()
	c := newHTTPContext()
	ex := exception.BadRequestException("bad")
	var called bool
	fn := traceHTTPCatch(ev, "Test.Filter", func(c *ctx.HTTPContext, ex *exception.Exception) { called = true })
	fn(c, &ex)
	if !called {
		t.Error(test.DiffMessage(false, true, "wrapped catch fn should still run when no listener is attached"))
	}
}

func TestTraceHTTPCatch_EmitsOnlyWhenListenerAttached(t *testing.T) {
	ev := event.NewEvent()
	c := newHTTPContext()
	ex := exception.BadRequestException("bad")
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		got = &te
	})
	fn := traceHTTPCatch(ev, "Test.Filter", func(c *ctx.HTTPContext, ex *exception.Exception) {})
	fn(c, &ex)
	if got == nil {
		t.Fatal("expected a trace event to be emitted when a listener is attached")
	}
	if got.Stage != trace.StageExceptionFilter || got.Name != "Test.Filter" {
		t.Error(test.DiffMessage([]any{got.Stage, got.Name}, []any{trace.StageExceptionFilter, "Test.Filter"}, "traceHTTPCatch emitted event fields"))
	}
}

func TestTraceWSHandler_NoListener_StillCallsWrapped(t *testing.T) {
	ev := event.NewEvent()
	c := ctx.NewWSContext()
	var called bool
	h := traceWSHandler(ev, trace.StageGuard, "Test.Guard", func(c *ctx.WSContext) { called = true })
	h(c)
	if !called {
		t.Error(test.DiffMessage(false, true, "wrapped handler should still run when no listener is attached"))
	}
}

func TestTraceWSHandler_EmitsOnlyWhenListenerAttached(t *testing.T) {
	ev := event.NewEvent()
	c := ctx.NewWSContext()
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		got = &te
	})
	h := traceWSHandler(ev, trace.StageGuard, "Test.Guard", func(c *ctx.WSContext) {})
	h(c)
	if got == nil {
		t.Fatal("expected a trace event to be emitted when a listener is attached")
	}
	if got.Stage != trace.StageGuard || got.Name != "Test.Guard" || got.Transport != trace.TransportWS {
		t.Error(test.DiffMessage([]any{got.Stage, got.Name, got.Transport}, []any{trace.StageGuard, "Test.Guard", trace.TransportWS}, "traceWSHandler emitted event fields"))
	}
}

func TestTraceWSCatch_NoListener_StillCallsWrapped(t *testing.T) {
	ev := event.NewEvent()
	c := ctx.NewWSContext()
	ex := exception.BadRequestException("bad")
	var called bool
	fn := traceWSCatch(ev, "Test.Filter", func(c *ctx.WSContext, ex *exception.Exception) { called = true })
	fn(c, &ex)
	if !called {
		t.Error(test.DiffMessage(false, true, "wrapped catch fn should still run when no listener is attached"))
	}
}

func TestTraceWSCatch_EmitsOnlyWhenListenerAttached(t *testing.T) {
	ev := event.NewEvent()
	c := ctx.NewWSContext()
	ex := exception.BadRequestException("bad")
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		got = &te
	})
	fn := traceWSCatch(ev, "Test.Filter", func(c *ctx.WSContext, ex *exception.Exception) {})
	fn(c, &ex)
	if got == nil {
		t.Fatal("expected a trace event to be emitted when a listener is attached")
	}
	if got.Stage != trace.StageExceptionFilter || got.Name != "Test.Filter" || got.Transport != trace.TransportWS {
		t.Error(test.DiffMessage([]any{got.Stage, got.Name, got.Transport}, []any{trace.StageExceptionFilter, "Test.Filter", trace.TransportWS}, "traceWSCatch emitted event fields"))
	}
}

func TestBuildUseMiddleware_NoListener_StillCallsNext(t *testing.T) {
	ev := event.NewEvent()
	c := newHTTPContext()
	var nextCalled bool
	c.Next = func() { nextCalled = true }
	mw := buildUseMiddleware(func(r *http.Request, w http.ResponseWriter, next ctx.Next) { next() }, ev, "Test.Middleware", trace.TransportHTTP)
	mw(c)
	if !nextCalled {
		t.Error(test.DiffMessage(false, true, "middleware should still call next when no listener is attached"))
	}
}

func TestBuildUseMiddleware_EmitsOnlyWhenListenerAttached(t *testing.T) {
	ev := event.NewEvent()
	c := newHTTPContext()
	c.Next = func() {}
	var got *trace.Event
	ev.On(trace.EventName, func(args ...any) {
		te := args[0].(trace.Event)
		got = &te
	})
	mw := buildUseMiddleware(func(r *http.Request, w http.ResponseWriter, next ctx.Next) { next() }, ev, "Test.Middleware", trace.TransportHTTP)
	mw(c)
	if got == nil {
		t.Fatal("expected a trace event to be emitted when a listener is attached")
	}
	if got.Stage != trace.StageMiddleware || got.Name != "Test.Middleware" || got.Transport != trace.TransportHTTP {
		t.Error(test.DiffMessage([]any{got.Stage, got.Name, got.Transport}, []any{trace.StageMiddleware, "Test.Middleware", trace.TransportHTTP}, "buildUseMiddleware emitted event fields"))
	}
}

type injectDepsTestProvider struct{}

func (injectDepsTestProvider) NewProvider() Provider { return injectDepsTestProvider{} }

type injectDepsComponent struct {
	Provider injectDepsTestProvider
	Plain    string
}

func TestInjectDependencies_ResolvesFromLocalProviders(t *testing.T) {
	p := injectDepsTestProvider{}
	injectedProviders := map[string]Provider{
		genFieldKey(reflect.TypeOf(p)): p,
	}
	component := injectDepsComponent{Plain: "kept"}
	result, err := injectDependencies(component, "provider", injectedProviders)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := result.Elem().Interface().(injectDepsComponent)
	if got.Plain != "kept" {
		t.Error(test.DiffMessage(got.Plain, "kept", "injectDependencies should pass through non-Provider fields unchanged"))
	}
}

type injectDepsUnresolvable struct {
	Missing unresolvableProviderType
}

type unresolvableProviderType struct{}

func (unresolvableProviderType) NewProvider() Provider { return unresolvableProviderType{} }

func TestInjectDependencies_UnresolvedProviderReturnsError(t *testing.T) {
	_, err := injectDependencies(injectDepsUnresolvable{}, "provider", map[string]Provider{})
	if err == nil {
		t.Error(test.DiffMessage(nil, "error", "injectDependencies should error when a Provider field can't be resolved"))
	}
}

func TestBuildFieldInjectionCallback_ResolvesFromLocalProviders(t *testing.T) {
	p := injectDepsTestProvider{}
	injectedProviders := map[string]Provider{
		genFieldKey(reflect.TypeOf(p)): p,
	}
	cb := buildFieldInjectionCallback("guarder", injectedProviders)

	ownerType := reflect.TypeOf(injectDepsComponent{})
	ownerValue := reflect.ValueOf(injectDepsComponent{Plain: "kept"})
	newInstance := reflect.New(ownerType)

	cb(0, ownerType, ownerValue, newInstance)
	cb(1, ownerType, ownerValue, newInstance)

	got := newInstance.Elem().Interface().(injectDepsComponent)
	if got.Provider != p {
		t.Error(test.DiffMessage(got.Provider, p, "buildFieldInjectionCallback should resolve the Provider field"))
	}
	if got.Plain != "kept" {
		t.Error(test.DiffMessage(got.Plain, "kept", "buildFieldInjectionCallback should pass through non-Provider fields unchanged"))
	}
}

func TestBuildFieldInjectionCallback_UnresolvedProviderPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error(test.DiffMessage(nil, "panic", "buildFieldInjectionCallback should panic when a Provider field can't be resolved"))
		}
	}()

	cb := buildFieldInjectionCallback("guarder", map[string]Provider{})
	ownerType := reflect.TypeOf(injectDepsUnresolvable{})
	ownerValue := reflect.ValueOf(injectDepsUnresolvable{})
	newInstance := reflect.New(ownerType)
	cb(0, ownerType, ownerValue, newInstance)
}

func TestLogBoostrapNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error(test.DiffMessage(r, nil, "logBoostrap should not panic"))
		}
	}()
	logBoostrap(8080)
}
