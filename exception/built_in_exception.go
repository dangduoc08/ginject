package exception

import (
	"net/http"
	"strconv"
)

func BadRequestException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusBadRequest), opts...)
}

func ConflictException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusConflict), opts...)
}

func ForbiddenException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusForbidden), opts...)
}

func GoneException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusGone), opts...)
}

func InternalServerErrorException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusInternalServerError), opts...)
}

func MethodNotAllowedException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusMethodNotAllowed), opts...)
}

func NotAcceptableException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusNotAcceptable), opts...)
}

func NotFoundException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusNotFound), opts...)
}

func RequestTimeoutException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusRequestTimeout), opts...)
}

func UnauthorizedException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusUnauthorized), opts...)
}

func RequestEntityTooLargeException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusRequestEntityTooLarge), opts...)
}

func UnsupportedMediaTypeException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusUnsupportedMediaType), opts...)
}

func UnprocessableEntityException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusUnprocessableEntity), opts...)
}

func NotImplementedException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusNotImplemented), opts...)
}

func HTTPVersionNotSupportedException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusHTTPVersionNotSupported), opts...)
}

func BadGatewayException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusBadGateway), opts...)
}

func ServiceUnavailableException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusServiceUnavailable), opts...)
}

func GatewayTimeoutException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusGatewayTimeout), opts...)
}

func TeapotException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusTeapot), opts...)
}

func PreconditionFailedException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusPreconditionFailed), opts...)
}

func MisdirectedRequestException(response string, opts ...any) HTTPException {
	return NewHTTPException(response, strconv.Itoa(http.StatusMisdirectedRequest), opts...)
}
