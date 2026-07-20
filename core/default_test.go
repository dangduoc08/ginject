package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/test"
)

func newHTTPContext() *ctx.HTTPContext {
	c := ctx.NewHTTPContext()
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.ResponseWriter = httptest.NewRecorder()
	return c
}

func TestGlobalExceptionFilterHTTPFullException(t *testing.T) {
	filter := globalExceptionFilter{}
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)

	ex := exception.BadRequestException("invalid input")
	filter.Catch(c, &ex)

	if w.Code != http.StatusBadRequest {
		t.Error(test.DiffMessage(w.Code, http.StatusBadRequest, "HTTP status for BadRequest"))
	}
}

func TestGlobalExceptionFilterHTTPStructMessage(t *testing.T) {
	filter := globalExceptionFilter{}
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)

	ex := exception.InternalServerErrorException(map[string]any{"detail": "db error"})
	filter.Catch(c, &ex)

	if w.Code != http.StatusInternalServerError {
		t.Error(test.DiffMessage(w.Code, http.StatusInternalServerError, "HTTP status"))
	}
}

func TestGlobalExceptionFilterHTTPIntMessage(t *testing.T) {
	filter := globalExceptionFilter{}
	c := newHTTPContext()

	ex := exception.InternalServerErrorException(42)
	filter.Catch(c, &ex)
}

func TestGlobalExceptionFilterHTTPNilMessage(t *testing.T) {
	filter := globalExceptionFilter{}
	c := newHTTPContext()

	ex := exception.InternalServerErrorException(nil)
	filter.Catch(c, &ex)
}
