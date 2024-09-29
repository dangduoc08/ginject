package exception

import (
	"net/http"
	"strconv"
)

func BadRequestException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusBadRequest), opts...)
}

func ConflictException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusConflict), opts...)
}

func ForbiddenException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusForbidden), opts...)
}

func GoneException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusGone), opts...)
}

func InternalServerErrorException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusInternalServerError), opts...)
}

func MethodNotAllowedException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusMethodNotAllowed), opts...)
}

func NotAcceptableException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusNotAcceptable), opts...)
}

func NotFoundException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusNotFound), opts...)
}

func RequestTimeoutException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusRequestTimeout), opts...)
}

func UnauthorizedException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusUnauthorized), opts...)
}

func RequestEntityTooLargeException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusRequestEntityTooLarge), opts...)
}

func UnsupportedMediaTypeException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusUnsupportedMediaType), opts...)
}

func UnprocessableEntityException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusUnprocessableEntity), opts...)
}

func NotImplementedException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusNotImplemented), opts...)
}

func HTTPVersionNotSupportedException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusHTTPVersionNotSupported), opts...)
}

func BadGatewayException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusBadGateway), opts...)
}

func ServiceUnavailableException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusServiceUnavailable), opts...)
}

func GatewayTimeoutException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusGatewayTimeout), opts...)
}

func TeapotException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusTeapot), opts...)
}

func PreconditionFailedException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusPreconditionFailed), opts...)
}

func MisdirectedRequestException(response any, opts ...any) Exception {
	return NewException(response, strconv.Itoa(http.StatusMisdirectedRequest), opts...)
}
