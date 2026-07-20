package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
)

func BenchmarkGlobalHTTPExceptionFilterFull(b *testing.B) {
	filter := globalHTTPExceptionFilter{}
	ex := exception.BadRequestException("validation error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := ctx.NewHTTPContext()
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.ResponseWriter = httptest.NewRecorder()
		filter.Catch(c, &ex)
	}
}

func BenchmarkGlobalHTTPExceptionFilterFallback(b *testing.B) {
	filter := globalHTTPExceptionFilter{}
	ex := exception.InternalServerErrorException("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := ctx.NewHTTPContext()
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.ResponseWriter = httptest.NewRecorder()
		filter.Catch(c, &ex)
	}
}
