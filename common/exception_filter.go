package common

import (
	"reflect"

	"github.com/dangduoc08/gogo/ctx"
	"github.com/dangduoc08/gogo/exception"
	"github.com/dangduoc08/gogo/routing"
)

type Catch = func(*ctx.Context, *exception.Exception)

type ExceptionFilterable interface {
	Catch(*ctx.Context, *exception.Exception)
}

type ExceptionFilterHandler struct {
	ExceptionFilterable ExceptionFilterable
	Handlers            []any
}

type RESTExceptionFilterItem struct {
	Method  string
	Route   string
	Version string
	Common  CommonItem
}

type WSExceptionFilterItem struct {
	EventName string
	Common    CommonItem
}

type ExceptionFilterItem struct {
	REST RESTExceptionFilterItem
	WS   WSExceptionFilterItem
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

func (e *ExceptionFilter) InjectProvidersIntoRESTExceptionFilters(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
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

		shouldAddExceptionFilter := map[string]bool{}
		for _, handler := range exceptionFilterHandler.Handlers {
			fnName := GetFnName(handler)
			if pattern, ok := r.FnNameToPatternMap[fnName]; ok {
				shouldAddExceptionFilter[pattern] = true
			}
		}

		for pattern := range r.PatternToFnNameMap {
			if _, ok := shouldAddExceptionFilter[pattern]; ok || len(shouldAddExceptionFilter) == 0 {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				exceptionFilterItemArr = append(exceptionFilterItemArr, ExceptionFilterItem{
					REST: RESTExceptionFilterItem{
						Method:  httpMethod,
						Route:   routing.ToEndpoint(route),
						Version: version,
						Common: CommonItem{
							Handler: exceptionFilterHandler.ExceptionFilterable.Catch,
							Name:    exceptionFilterableType.String(),
						},
					},
				})
			}
		}
	}

	return exceptionFilterItemArr
}

func (e *ExceptionFilter) InjectProvidersIntoWSExceptionFilters(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
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

		shouldAddExceptionFilter := map[string]bool{}
		for _, handler := range exceptionFilterHandler.Handlers {
			fnName := GetFnName(handler)
			_, eventName, _ := ParseFnNameToURL(fnName, WSOperations)
			eventName = ToWSEventName(ws.subprotocol, eventName)
			shouldAddExceptionFilter[eventName] = true
		}

		for pattern := range ws.patternToFnNameMap {
			if _, ok := shouldAddExceptionFilter[pattern]; ok || len(shouldAddExceptionFilter) == 0 {
				exceptionFilterItemArr = append(exceptionFilterItemArr, ExceptionFilterItem{
					WS: WSExceptionFilterItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: exceptionFilterHandler.ExceptionFilterable.Catch,
							Name:    exceptionFilterableType.String(),
						},
					},
				})
			}
		}
	}

	return exceptionFilterItemArr
}
