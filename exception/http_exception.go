package exception

import (
	"net/http"
	"strconv"
)

var (
	codeBadRequest                = strconv.Itoa(http.StatusBadRequest)
	codeConflict                  = strconv.Itoa(http.StatusConflict)
	codeForbidden                 = strconv.Itoa(http.StatusForbidden)
	codeGone                      = strconv.Itoa(http.StatusGone)
	codeInternalServerError       = strconv.Itoa(http.StatusInternalServerError)
	codeMethodNotAllowed          = strconv.Itoa(http.StatusMethodNotAllowed)
	codeNotAcceptable             = strconv.Itoa(http.StatusNotAcceptable)
	codeNotFound                  = strconv.Itoa(http.StatusNotFound)
	codeRequestTimeout            = strconv.Itoa(http.StatusRequestTimeout)
	codeUnauthorized              = strconv.Itoa(http.StatusUnauthorized)
	codeRequestEntityTooLarge     = strconv.Itoa(http.StatusRequestEntityTooLarge)
	codeUnsupportedMediaType      = strconv.Itoa(http.StatusUnsupportedMediaType)
	codeUnprocessableEntity       = strconv.Itoa(http.StatusUnprocessableEntity)
	codeNotImplemented            = strconv.Itoa(http.StatusNotImplemented)
	codeHTTPVersionNotSupported   = strconv.Itoa(http.StatusHTTPVersionNotSupported)
	codeBadGateway                = strconv.Itoa(http.StatusBadGateway)
	codeServiceUnavailable        = strconv.Itoa(http.StatusServiceUnavailable)
	codeGatewayTimeout            = strconv.Itoa(http.StatusGatewayTimeout)
	codeTeapot                    = strconv.Itoa(http.StatusTeapot)
	codePreconditionFailed        = strconv.Itoa(http.StatusPreconditionFailed)
	codeMisdirectedRequest        = strconv.Itoa(http.StatusMisdirectedRequest)
)

func BadRequestException(response any, opts ...any) Exception {
	return NewException(response, codeBadRequest, opts...)
}

func ConflictException(response any, opts ...any) Exception {
	return NewException(response, codeConflict, opts...)
}

func ForbiddenException(response any, opts ...any) Exception {
	return NewException(response, codeForbidden, opts...)
}

func GoneException(response any, opts ...any) Exception {
	return NewException(response, codeGone, opts...)
}

func InternalServerErrorException(response any, opts ...any) Exception {
	return NewException(response, codeInternalServerError, opts...)
}

func MethodNotAllowedException(response any, opts ...any) Exception {
	return NewException(response, codeMethodNotAllowed, opts...)
}

func NotAcceptableException(response any, opts ...any) Exception {
	return NewException(response, codeNotAcceptable, opts...)
}

func NotFoundException(response any, opts ...any) Exception {
	return NewException(response, codeNotFound, opts...)
}

func RequestTimeoutException(response any, opts ...any) Exception {
	return NewException(response, codeRequestTimeout, opts...)
}

func UnauthorizedException(response any, opts ...any) Exception {
	return NewException(response, codeUnauthorized, opts...)
}

func RequestEntityTooLargeException(response any, opts ...any) Exception {
	return NewException(response, codeRequestEntityTooLarge, opts...)
}

func UnsupportedMediaTypeException(response any, opts ...any) Exception {
	return NewException(response, codeUnsupportedMediaType, opts...)
}

func UnprocessableEntityException(response any, opts ...any) Exception {
	return NewException(response, codeUnprocessableEntity, opts...)
}

func NotImplementedException(response any, opts ...any) Exception {
	return NewException(response, codeNotImplemented, opts...)
}

func HTTPVersionNotSupportedException(response any, opts ...any) Exception {
	return NewException(response, codeHTTPVersionNotSupported, opts...)
}

func BadGatewayException(response any, opts ...any) Exception {
	return NewException(response, codeBadGateway, opts...)
}

func ServiceUnavailableException(response any, opts ...any) Exception {
	return NewException(response, codeServiceUnavailable, opts...)
}

func GatewayTimeoutException(response any, opts ...any) Exception {
	return NewException(response, codeGatewayTimeout, opts...)
}

func TeapotException(response any, opts ...any) Exception {
	return NewException(response, codeTeapot, opts...)
}

func PreconditionFailedException(response any, opts ...any) Exception {
	return NewException(response, codePreconditionFailed, opts...)
}

func MisdirectedRequestException(response any, opts ...any) Exception {
	return NewException(response, codeMisdirectedRequest, opts...)
}
