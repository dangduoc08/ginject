package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

func BenchmarkGlobalExceptionFilterFull(b *testing.B) {
	filter := globalExceptionFilter{}
	ex := exception.BadRequestException("validation error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := ctx.NewContext()
		c.Broker = broker.NewWithConfig(broker.Config{RecoverPanics: true})
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.ResponseWriter = httptest.NewRecorder()
		c.Type = ctx.HTTPType
		filter.Catch(c, &ex)
	}
}

func BenchmarkGlobalExceptionFilterFallback(b *testing.B) {
	filter := globalExceptionFilter{}
	ex := exception.InternalServerErrorException(42)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := ctx.NewContext()
		c.Broker = broker.NewWithConfig(broker.Config{RecoverPanics: true})
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.ResponseWriter = httptest.NewRecorder()
		c.Type = ctx.HTTPType
		filter.Catch(c, &ex)
	}
}
