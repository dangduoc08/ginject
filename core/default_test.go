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

func TestGlobalHTTPExceptionFilterFullException(t *testing.T) {
	filter := globalHTTPExceptionFilter{}
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)

	ex := exception.BadRequestException("invalid input")
	filter.Catch(c, &ex)

	if w.Code != http.StatusBadRequest {
		t.Error(test.DiffMessage(w.Code, http.StatusBadRequest, "HTTP status for BadRequest"))
	}
}

func TestGlobalHTTPExceptionFilterEmptyMessageFallsBackToDefault(t *testing.T) {
	filter := globalHTTPExceptionFilter{}
	c := newHTTPContext()
	w := c.ResponseWriter.(*httptest.ResponseRecorder)

	ex := exception.InternalServerErrorException("")
	filter.Catch(c, &ex)

	if w.Code != http.StatusInternalServerError {
		t.Error(test.DiffMessage(w.Code, http.StatusInternalServerError, "HTTP status"))
	}
}

func TestGlobalWSExceptionFilterFullException(t *testing.T) {
	filter := globalWSExceptionFilter{}
	c := ctx.NewWSContext()

	var got any
	c.SetSend(func(data any) { got = data })

	ex := exception.PolicyViolationException("nope")
	filter.Catch(c, &ex)

	env, ok := got.(ctx.Map)
	if !ok {
		t.Fatalf("expected ctx.Map envelope, got %T: %+v", got, got)
	}
	if env["code"] != 1008 {
		t.Error(test.DiffMessage(env["code"], 1008, "WS status code for PolicyViolation"))
	}
	if env["error"] != "Policy Violation" {
		t.Error(test.DiffMessage(env["error"], "Policy Violation", "WS status text"))
	}
	if env["message"] != "nope" {
		t.Error(test.DiffMessage(env["message"], "nope", "WS message"))
	}
}

func TestGlobalWSExceptionFilterEmptyMessageFallsBackToDefault(t *testing.T) {
	filter := globalWSExceptionFilter{}
	c := ctx.NewWSContext()

	var got any
	c.SetSend(func(data any) { got = data })

	ex := exception.InternalServerErrorException("")
	filter.Catch(c, &ex)

	env, ok := got.(ctx.Map)
	if !ok {
		t.Fatalf("expected ctx.Map envelope, got %T: %+v", got, got)
	}
	if env["code"] != http.StatusInternalServerError {
		t.Error(test.DiffMessage(env["code"], http.StatusInternalServerError, "WS status code fallback"))
	}
}
