package core

import (
	"reflect"
	"testing"
)

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
