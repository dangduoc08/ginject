package core

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/ctx"
)

func BenchmarkServeHTTP(b *testing.B) {
	app := New()
	app.Create(ModuleBuilder().Build())

	r := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.ServeHTTP(w, r)
	}
}

func BenchmarkProvideAndInvoke(b *testing.B) {
	app := New()
	app.Create(ModuleBuilder().Build())

	c := ctx.NewContext()
	c.Broker = broker.NewWithConfig(broker.Config{RecoverPanics: true})
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	handler := func() reflect.Value { return reflect.ValueOf("bench") }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		invokeHandlerByProviders(handler, app.injectedProviders, c)
	}
}
