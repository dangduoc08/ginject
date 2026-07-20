package core

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

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
