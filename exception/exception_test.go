package exception

import (
	"errors"
	"net/http"
	"strconv"
	"testing"

	"github.com/dangduoc08/ginject/testutils"
)

func TestNewException_DefaultErrorFromCode(t *testing.T) {
	e := NewException("body", strconv.Itoa(http.StatusBadRequest))
	if e.Error() != "Bad Request" {
		t.Error(testutils.DiffMessage(e.Error(), "Bad Request", "default error from HTTP status"))
	}
}

func TestNewException_StringOpt(t *testing.T) {
	e := NewException("body", strconv.Itoa(http.StatusBadRequest), "custom message")
	if e.Error() != "custom message" {
		t.Error(testutils.DiffMessage(e.Error(), "custom message", "string opt overrides error"))
	}
}

func TestNewException_ErrorOpt(t *testing.T) {
	cause := errors.New("underlying error")
	e := NewException("body", strconv.Itoa(http.StatusBadRequest), cause)
	if e.Error() != "underlying error" {
		t.Error(testutils.DiffMessage(e.Error(), "underlying error", "error opt sets error"))
	}
}

func TestNewException_ExceptionOpt(t *testing.T) {
	inner := NewException("inner", strconv.Itoa(http.StatusNotFound))
	e := NewException("body", strconv.Itoa(http.StatusBadRequest), inner)
	if e.Error() != "Not Found" {
		t.Error(testutils.DiffMessage(e.Error(), "Not Found", "Exception opt copies error"))
	}
}

func TestNewException_ExceptionOptions_Description(t *testing.T) {
	e := NewException("body", strconv.Itoa(http.StatusBadRequest), ExceptionOptions{
		Description: "validation failed",
	})
	if e.Error() != "validation failed" {
		t.Error(testutils.DiffMessage(e.Error(), "validation failed", "ExceptionOptions description"))
	}
}

func TestNewException_ExceptionOptions_Cause(t *testing.T) {
	cause := errors.New("db error")
	e := NewException("body", strconv.Itoa(http.StatusInternalServerError), ExceptionOptions{
		Cause: cause,
	})
	if !errors.Is(e.error, cause) {
		t.Error(testutils.DiffMessage(errors.Is(e.error, cause), true, "cause is wrapped"))
	}
}

func TestNewException_ExceptionOptions_DescriptionAndCause(t *testing.T) {
	cause := errors.New("root cause")
	e := NewException("body", strconv.Itoa(http.StatusBadRequest), ExceptionOptions{
		Description: "custom",
		Cause:       cause,
	})
	if !errors.Is(e.error, cause) {
		t.Error(testutils.DiffMessage(errors.Is(e.error, cause), true, "cause wrapped under description"))
	}
	if e.Error() == "" {
		t.Error(testutils.DiffMessage(e.Error(), "non-empty", "error string not empty"))
	}
}

func TestNewException_InvalidCode(t *testing.T) {
	e := NewException("body", "not-a-number")
	code, text := e.GetHTTPStatus()
	if code != 0 || text != "" {
		t.Error(testutils.DiffMessage([]any{code, text}, []any{0, ""}, "invalid code returns zero"))
	}
}

func TestNewException_GetCode(t *testing.T) {
	e := NewException("body", "999")
	if e.GetCode() != "999" {
		t.Error(testutils.DiffMessage(e.GetCode(), "999", "GetCode returns raw code"))
	}
}

func TestNewException_GetResponse(t *testing.T) {
	e := NewException(map[string]string{"msg": "err"}, strconv.Itoa(http.StatusBadRequest))
	if e.GetResponse().(map[string]string)["msg"] != "err" {
		t.Error(testutils.DiffMessage(e.GetResponse(), map[string]string{"msg": "err"}, "GetResponse"))
	}
}

func TestException_GetHTTPStatus_Valid(t *testing.T) {
	e := NewException("body", strconv.Itoa(http.StatusNotFound))
	code, text := e.GetHTTPStatus()
	if code != http.StatusNotFound || text != "Not Found" {
		t.Error(testutils.DiffMessage([]any{code, text}, []any{404, "Not Found"}, "GetHTTPStatus valid"))
	}
}

func TestException_GetHTTPStatus_Unknown(t *testing.T) {
	e := NewException("body", "999")
	code, text := e.GetHTTPStatus()
	if code != 0 || text != "" {
		t.Error(testutils.DiffMessage([]any{code, text}, []any{0, ""}, "GetHTTPStatus unknown code"))
	}
}

func TestException_Unwrap(t *testing.T) {
	cause := errors.New("root")
	e := NewException("body", strconv.Itoa(http.StatusBadRequest), ExceptionOptions{Cause: cause})
	if !errors.Is(e, cause) {
		t.Error(testutils.DiffMessage(errors.Is(e, cause), true, "Unwrap chains to cause"))
	}
}

func TestHTTPExceptionHelpers(t *testing.T) {
	cases := []struct {
		fn     func(any, ...any) Exception
		status int
	}{
		{BadRequestException, http.StatusBadRequest},
		{ConflictException, http.StatusConflict},
		{ForbiddenException, http.StatusForbidden},
		{GoneException, http.StatusGone},
		{InternalServerErrorException, http.StatusInternalServerError},
		{MethodNotAllowedException, http.StatusMethodNotAllowed},
		{NotAcceptableException, http.StatusNotAcceptable},
		{NotFoundException, http.StatusNotFound},
		{RequestTimeoutException, http.StatusRequestTimeout},
		{UnauthorizedException, http.StatusUnauthorized},
		{RequestEntityTooLargeException, http.StatusRequestEntityTooLarge},
		{UnsupportedMediaTypeException, http.StatusUnsupportedMediaType},
		{UnprocessableEntityException, http.StatusUnprocessableEntity},
		{NotImplementedException, http.StatusNotImplemented},
		{HTTPVersionNotSupportedException, http.StatusHTTPVersionNotSupported},
		{BadGatewayException, http.StatusBadGateway},
		{ServiceUnavailableException, http.StatusServiceUnavailable},
		{GatewayTimeoutException, http.StatusGatewayTimeout},
		{TeapotException, http.StatusTeapot},
		{PreconditionFailedException, http.StatusPreconditionFailed},
		{MisdirectedRequestException, http.StatusMisdirectedRequest},
	}
	for _, c := range cases {
		e := c.fn("body")
		code, _ := e.GetHTTPStatus()
		if code != c.status {
			t.Error(testutils.DiffMessage(code, c.status, "HTTP status helper"))
		}
	}
}
