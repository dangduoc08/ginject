package common

import (
	"reflect"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/exception"
	"github.com/dangduoc08/gooh/routing"
)

type Catch = func(*context.Context, *exception.HTTPException)

type ExceptionFilterable interface {
	Catch(*context.Context, *exception.HTTPException)
}

type ExceptionFilterHandler struct {
	ExceptionFilterable ExceptionFilterable
	Handlers            []any
}

type ExceptionFilterItem struct {
	Method  string
	Route   string
	Handler any
}

type ExceptionFilter struct {
	ExceptionFilterHandlers []ExceptionFilterHandler
}

func (e *ExceptionFilter) BindExceptionFilter(exceptionFilterable ExceptionFilterable, handlers ...any) *ExceptionFilter {
	exceptionFilterHandler := ExceptionFilterHandler{
		ExceptionFilterable: exceptionFilterable,
		Handlers:            handlers,
	}

	e.ExceptionFilterHandlers = append(e.ExceptionFilterHandlers, exceptionFilterHandler)
	return e
}

func (e *ExceptionFilter) AddExceptionFiltersToModule(r *Rest, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
	exceptionFilterItemArr := []ExceptionFilterItem{}

	for _, exceptionFilterHandler := range e.ExceptionFilterHandlers {

		exceptionFilterableType := reflect.TypeOf(exceptionFilterHandler.ExceptionFilterable)
		exceptionFilterableValue := reflect.ValueOf(exceptionFilterHandler.ExceptionFilterable)
		newExceptionFilter := reflect.New(exceptionFilterableType)

		for i := 0; i < exceptionFilterableType.NumField(); i++ {

			// callback use to inject providers
			cb(i, exceptionFilterableType, exceptionFilterableValue, newExceptionFilter)
		}

		// invoke exceptionFilter constructor
		// if NewExceptionFilter was declared
		newExceptionFilterable := newExceptionFilter.Interface()
		newExceptionFilterable = Construct(newExceptionFilterable, "NewExceptionFilter")

		exceptionFilterHandler.ExceptionFilterable = newExceptionFilterable.(ExceptionFilterable)

		for pattern := range r.patternToFnNameMap {
			httpMethod, route := routing.SplitRoute(pattern)
			exceptionFilterItemArr = append(exceptionFilterItemArr, ExceptionFilterItem{
				Method:  httpMethod,
				Route:   routing.ToEndpoint(route),
				Handler: exceptionFilterHandler.ExceptionFilterable.Catch,
			})
		}
	}

	return exceptionFilterItemArr
}
