package exception

import (
	"net/http"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestHTTPExceptionHelpers(t *testing.T) {
	cases := []struct {
		fn     func(string, ...any) Exception
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
		{TooManyRequestsException, http.StatusTooManyRequests},
	}
	for _, c := range cases {
		e := c.fn("body")
		if e.GetCode() != c.status {
			t.Error(test.DiffMessage(e.GetCode(), c.status, "HTTP status helper code"))
		}
		if e.GetStatusText() != http.StatusText(c.status) {
			t.Error(test.DiffMessage(e.GetStatusText(), http.StatusText(c.status), "HTTP status helper text"))
		}
	}
}
