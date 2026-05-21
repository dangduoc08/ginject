package core

import (
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/testutils"
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
			t.Error(testutils.DiffMessage(got, c.want, "getPkgFromControllerKey("+c.in+")"))
		}
	}
}

func TestGenFieldKey(t *testing.T) {
	t1 := reflect.TypeOf(mockProvider{})
	got := genFieldKey(t1)
	want := t1.PkgPath() + "/" + t1.String()
	if got != want {
		t.Error(testutils.DiffMessage(got, want, "genFieldKey"))
	}
}

func TestGenProviderKey(t *testing.T) {
	p := &mockProvider{}
	got := genProviderKey(p)
	want := genFieldKey(reflect.TypeOf(p))
	if got != want {
		t.Error(testutils.DiffMessage(got, want, "genProviderKey"))
	}
}

func TestGenControllerKey(t *testing.T) {
	m := ModuleBuilder().Build()
	c := &mockController{}
	got := genControllerKey(m, c)
	want := "[" + m.ID() + "]" + genFieldKey(reflect.TypeOf(c))
	if got != want {
		t.Error(testutils.DiffMessage(got, want, "genControllerKey"))
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
			t.Error(testutils.DiffMessage(got, c.want, "isDynamicModule("+c.in+")"))
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
		t.Error(testutils.DiffMessage(len(controllers), 1, "toUniqueControllers dedup"))
	}
}

func TestToUniqueControllersEmpty(t *testing.T) {
	m := ModuleBuilder().Build()
	controllers := []Controller{}
	toUniqueControllers(m, &controllers)
	if len(controllers) != 0 {
		t.Error(testutils.DiffMessage(len(controllers), 0, "toUniqueControllers empty"))
	}
}

func TestIsInjectedProvider(t *testing.T) {
	providerType := reflect.TypeOf(mockProvider{})
	notProviderType := reflect.TypeOf(struct{}{})

	if !isInjectedProvider(providerType) {
		t.Error(testutils.DiffMessage(false, true, "mockProvider should be injectable"))
	}
	if isInjectedProvider(notProviderType) {
		t.Error(testutils.DiffMessage(true, false, "anonymous struct should not be injectable"))
	}
}
