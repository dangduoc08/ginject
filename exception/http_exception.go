package exception

import (
	"net/http"
)

const (
	codeBadRequest              = http.StatusBadRequest
	codeConflict                = http.StatusConflict
	codeForbidden               = http.StatusForbidden
	codeGone                    = http.StatusGone
	codeInternalServerError     = http.StatusInternalServerError
	codeMethodNotAllowed        = http.StatusMethodNotAllowed
	codeNotAcceptable           = http.StatusNotAcceptable
	codeNotFound                = http.StatusNotFound
	codeRequestTimeout          = http.StatusRequestTimeout
	codeUnauthorized            = http.StatusUnauthorized
	codeRequestEntityTooLarge   = http.StatusRequestEntityTooLarge
	codeUnsupportedMediaType    = http.StatusUnsupportedMediaType
	codeUnprocessableEntity     = http.StatusUnprocessableEntity
	codeNotImplemented          = http.StatusNotImplemented
	codeHTTPVersionNotSupported = http.StatusHTTPVersionNotSupported
	codeBadGateway              = http.StatusBadGateway
	codeServiceUnavailable      = http.StatusServiceUnavailable
	codeGatewayTimeout          = http.StatusGatewayTimeout
	codeTeapot                  = http.StatusTeapot
	codePreconditionFailed      = http.StatusPreconditionFailed
	codeMisdirectedRequest      = http.StatusMisdirectedRequest
	codeTooManyRequests         = http.StatusTooManyRequests
)

func BadRequestException(message string, opts ...any) Exception {
	return NewException(message, codeBadRequest, opts...)
}

func ConflictException(message string, opts ...any) Exception {
	return NewException(message, codeConflict, opts...)
}

func ForbiddenException(message string, opts ...any) Exception {
	return NewException(message, codeForbidden, opts...)
}

func GoneException(message string, opts ...any) Exception {
	return NewException(message, codeGone, opts...)
}

func InternalServerErrorException(message string, opts ...any) Exception {
	return NewException(message, codeInternalServerError, opts...)
}

func MethodNotAllowedException(message string, opts ...any) Exception {
	return NewException(message, codeMethodNotAllowed, opts...)
}

func NotAcceptableException(message string, opts ...any) Exception {
	return NewException(message, codeNotAcceptable, opts...)
}

func NotFoundException(message string, opts ...any) Exception {
	return NewException(message, codeNotFound, opts...)
}

func RequestTimeoutException(message string, opts ...any) Exception {
	return NewException(message, codeRequestTimeout, opts...)
}

func UnauthorizedException(message string, opts ...any) Exception {
	return NewException(message, codeUnauthorized, opts...)
}

func RequestEntityTooLargeException(message string, opts ...any) Exception {
	return NewException(message, codeRequestEntityTooLarge, opts...)
}

func UnsupportedMediaTypeException(message string, opts ...any) Exception {
	return NewException(message, codeUnsupportedMediaType, opts...)
}

func UnprocessableEntityException(message string, opts ...any) Exception {
	return NewException(message, codeUnprocessableEntity, opts...)
}

func NotImplementedException(message string, opts ...any) Exception {
	return NewException(message, codeNotImplemented, opts...)
}

func HTTPVersionNotSupportedException(message string, opts ...any) Exception {
	return NewException(message, codeHTTPVersionNotSupported, opts...)
}

func BadGatewayException(message string, opts ...any) Exception {
	return NewException(message, codeBadGateway, opts...)
}

func ServiceUnavailableException(message string, opts ...any) Exception {
	return NewException(message, codeServiceUnavailable, opts...)
}

func GatewayTimeoutException(message string, opts ...any) Exception {
	return NewException(message, codeGatewayTimeout, opts...)
}

func TeapotException(message string, opts ...any) Exception {
	return NewException(message, codeTeapot, opts...)
}

func PreconditionFailedException(message string, opts ...any) Exception {
	return NewException(message, codePreconditionFailed, opts...)
}

func MisdirectedRequestException(message string, opts ...any) Exception {
	return NewException(message, codeMisdirectedRequest, opts...)
}

func TooManyRequestsException(message string, opts ...any) Exception {
	return NewException(message, codeTooManyRequests, opts...)
}
