package common

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/internal/color"
)

const InterceptorMethodName = "Intercept"

type WithValueKey string

type Interceptable any

type InterceptorItem struct {
	REST RESTInterceptorItem
	WS   WSInterceptorItem
}

type interceptorHandler struct {
	interceptable Interceptable
	handlers      []any
}

type Interceptor struct {
	InterceptorHandlers []interceptorHandler
}

func (i *Interceptor) BindInterceptor(interceptable Interceptable, handlers ...any) *Interceptor {
	interceptorHandler := interceptorHandler{
		interceptable: interceptable,
		handlers:      handlers,
	}

	i.InterceptorHandlers = append(i.InterceptorHandlers, interceptorHandler)

	return i
}

func InterceptorShapeError(interceptable any) error {
	return errors.New(color.FmtRed(
		"invalid interceptor: %v has no %s method usable as an interceptor",
		reflect.TypeOf(interceptable),
		InterceptorMethodName,
	))
}
