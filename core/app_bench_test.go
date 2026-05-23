package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

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

func BenchmarkPublishWSEvent(b *testing.B) {
	app := New()
	for i := 0; i < 50; i++ {
		app.wsEventToID[fmt.Sprintf("other:%d", i)] = []string{"wsid"}
	}
	const target = "target-event"
	const wsid = "conn-1"
	app.wsEventToID[target] = []string{wsid}

	c := ctx.NewContext()
	c.Event = ctx.NewEvent()
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Event.On(target+wsid, func(args ...any) {})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.publishWSEvent(target, "hello", c)
	}
}

func BenchmarkProvideAndInvoke(b *testing.B) {
	app := New()
	app.Create(ModuleBuilder().Build())

	c := ctx.NewContext()
	c.Event = ctx.NewEvent()
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	handler := func() reflect.Value { return reflect.ValueOf("bench") }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		app.provideAndInvoke(handler, c)
	}
}
