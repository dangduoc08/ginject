package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

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
	got := getDependency(contextKey, c, reflect.Value{})
	if got != c {
		t.Error(test.DiffMessage(got, c, "getDependency contextKey"))
	}
}

func TestGetDependencyRequest(t *testing.T) {
	c := newHTTPContext()
	got := getDependency(requestKey, c, reflect.Value{})
	if got != c.Request {
		t.Error(test.DiffMessage(got, c.Request, "getDependency requestKey"))
	}
}

func TestGetDependencyResponse(t *testing.T) {
	c := newHTTPContext()
	got := getDependency(responseKey, c, reflect.Value{})
	if got != c.ResponseWriter {
		t.Error(test.DiffMessage(got, c.ResponseWriter, "getDependency responseKey"))
	}
}

func TestGetDependencyUnknownReturnsDependencies(t *testing.T) {
	c := newHTTPContext()
	got := getDependency("unknown-key", c, reflect.Value{})
	if got == nil {
		t.Error(test.DiffMessage(nil, "dependencies map", "getDependency unknown key should return dependencies"))
	}
}

func TestIsInjectableHandlerValid(t *testing.T) {
	handler := func(c *ctx.Context) {}
	err := isInjectableHandler(handler, nil)
	if err != nil {
		t.Error(test.DiffMessage(err, nil, "isInjectableHandler valid handler"))
	}
}

func TestIsInjectableHandlerInvalid(t *testing.T) {
	type unknownType struct{}
	handler := func(_ unknownType) {}
	err := isInjectableHandler(handler, nil)
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
