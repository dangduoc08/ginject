package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/common"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/trace"
)

type benchPlainController struct {
	common.HTTP
}

func (c benchPlainController) NewController() Controller { return c }
func (c benchPlainController) READ_bench() string         { return "ok" }

func benchHTTPContext(urlPath string) *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, urlPath, nil)
	c.ResponseWriter = httptest.NewRecorder()
	return c
}

func BenchmarkHandleRequest_MatchedRoute(b *testing.B) {
	resetModuleGlobals()
	app := New()
	app.Create(ModuleBuilder().Controllers(benchPlainController{}).Build())

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c := benchHTTPContext("/bench")
		app.http.handleRequest(c)
	}
}

func BenchmarkHandleRequest_NotFound(b *testing.B) {
	resetModuleGlobals()
	app := New()
	app.Create(ModuleBuilder().Controllers(benchPlainController{}).Build())

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c := benchHTTPContext("/does-not-exist")
		app.http.handleRequest(c)
	}
}

func BenchmarkHandleRequest_FullPipeline(b *testing.B) {
	resetModuleGlobals()
	app := New()
	app.Create(ModuleBuilder().Controllers(tracePipelineController{}).Build())

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c := benchHTTPContext("/tracepipeline")
		app.http.handleRequest(c)
	}
}

func BenchmarkHandleRequest_MultiInterceptor_NoListener(b *testing.B) {
	resetModuleGlobals()
	app := New()
	app.Create(ModuleBuilder().Controllers(multiInterceptorController{}).Build())

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c := benchHTTPContext("/multiintercept")
		app.http.handleRequest(c)
	}
}

func BenchmarkHandleRequest_MultiInterceptor_WithListener(b *testing.B) {
	resetModuleGlobals()
	app := New()
	app.event.On(trace.EventName, func(args ...any) {})
	app.Create(ModuleBuilder().Controllers(multiInterceptorController{}).Build())

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		c := benchHTTPContext("/multiintercept")
		app.http.handleRequest(c)
	}
}
