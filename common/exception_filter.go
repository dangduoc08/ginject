package common

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/color"
)

const ExceptionFilterMethodName = "Catch"

type ExceptionFilterable any

type ExceptionFilterItem struct {
	REST RESTExceptionFilterItem
	WS   WSExceptionFilterItem
}

type exceptionFilterHandler struct {
	exceptionFilterable ExceptionFilterable
	handlers            []any
}

type ExceptionFilter struct {
	ExceptionFilterHandlers []exceptionFilterHandler
}

func (e *ExceptionFilter) BindExceptionFilter(exceptionFilterable ExceptionFilterable, handlers ...any) *ExceptionFilter {
	exceptionFilterHandler := exceptionFilterHandler{
		exceptionFilterable: exceptionFilterable,
		handlers:            handlers,
	}

	e.ExceptionFilterHandlers = append(e.ExceptionFilterHandlers, exceptionFilterHandler)
	return e
}

func ExceptionFilterShapeError(exceptionFilterable any) error {
	return errors.New(color.FmtRed(
		"invalid exception filter: %v has no %s method usable as an exception filter",
		reflect.TypeOf(exceptionFilterable),
		ExceptionFilterMethodName,
	))
}

// ReqCtx holds either *ctx.HTTPContext or *ctx.WSContext depending on
// which transport's catch-chain published this payload.
type CatchEventPayload struct {
	ReqCtx    any
	Recovered any
	Index     int
}

func NormalizeRecovered(rec any) *exception.Exception {
	if ex, ok := rec.(exception.Exception); ok {
		return &ex
	}

	response := http.StatusText(http.StatusInternalServerError)
	switch arg := rec.(type) {
	case error:
		response = arg.Error()
	case string:
		response = arg
	}

	ex := exception.InternalServerErrorException(response, map[string]any{
		"description": "Unknown exception",
	})
	return &ex
}
