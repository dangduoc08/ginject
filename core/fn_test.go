package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/test"
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

func TestReturnRESTString(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnREST(c, reflect.ValueOf("hello"))
	if w.Body.String() != "hello" {
		t.Error(test.DiffMessage(w.Body.String(), "hello", "returnREST string"))
	}
}

func TestReturnRESTMap(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnREST(c, reflect.ValueOf(map[string]any{"k": "v"}))
	if w.Body.Len() == 0 {
		t.Error(test.DiffMessage(0, ">0", "returnREST map should produce JSON body"))
	}
}

func TestReturnRESTInt(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnREST(c, reflect.ValueOf(42))
	if w.Body.String() != "42" {
		t.Error(test.DiffMessage(w.Body.String(), "42", "returnREST int"))
	}
}

func TestReturnRESTBool(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnREST(c, reflect.ValueOf(true))
	if w.Body.String() != "true" {
		t.Error(test.DiffMessage(w.Body.String(), "true", "returnREST bool"))
	}
}

func TestReturnRESTSlice(t *testing.T) {
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)
	returnREST(c, reflect.ValueOf([]int{1, 2, 3}))
	if w.Body.Len() == 0 {
		t.Error(test.DiffMessage(0, ">0", "returnREST slice should produce JSON body"))
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

func TestIsInjectableHandlerValid(t *testing.T) {
	handler := func(c *ctx.HTTPContext) {}
	err := isInjectableHandler(handler, nil, knownRESTDependencyKeys)
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "isInjectableHandler valid handler"))
	}
}

func TestIsInjectableHandlerInvalid(t *testing.T) {
	type unknownType struct{}
	handler := func(_ unknownType) {}
	err := isInjectableHandler(handler, nil, knownRESTDependencyKeys)
	if err == nil {
		t.Error(test.DiffMessage(nil, "error", "isInjectableHandler with unknown arg should return error"))
	}
}

func TestLogBoostrapNoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Error(test.DiffMessage(r, nil, "logBoostrap should not panic"))
		}
	}()
	logBoostrap(8080)
}
