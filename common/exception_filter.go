package common

import (
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/routing"
)

type Catch = func(*ctx.Context, *exception.Exception)

type ExceptionFilterable interface {
	Catch(*ctx.Context, *exception.Exception)
}

type RESTExceptionFilterItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
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

func (e *ExceptionFilter) InjectProvidersIntoRESTExceptionFilters(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
	exceptionFilterItemArr := make([]ExceptionFilterItem, 0, len(r.PatternToFnNameMap)*len(e.ExceptionFilterHandlers))

	for _, exceptionFilterHandler := range e.ExceptionFilterHandlers {
		exceptionFilterableType := reflect.TypeOf(exceptionFilterHandler.exceptionFilterable)
		exceptionFilterableValue := reflect.ValueOf(exceptionFilterHandler.exceptionFilterable)
		newExceptionFilter := reflect.New(exceptionFilterableType)

		for i := 0; i < exceptionFilterableType.NumField(); i++ {
			cb(i, exceptionFilterableType, exceptionFilterableValue, newExceptionFilter)
		}

		newExceptionFilterable := newExceptionFilter.Interface()
		newExceptionFilterable = Construct(newExceptionFilterable, "NewExceptionFilter")
		exceptionFilterHandler.exceptionFilterable = newExceptionFilterable.(ExceptionFilterable)

		shouldAddExceptionFilter := map[string]bool{}
		for _, handler := range exceptionFilterHandler.handlers {
			fnName := GetFnName(handler)
			if pattern, ok := r.FnNameToPatternMap[fnName]; ok {
				shouldAddExceptionFilter[pattern] = true
			}
		}
		applyAll := len(shouldAddExceptionFilter) == 0

		for pattern := range r.PatternToFnNameMap {
			if applyAll || shouldAddExceptionFilter[pattern] {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				exceptionFilterItemArr = append(exceptionFilterItemArr, ExceptionFilterItem{
					REST: RESTExceptionFilterItem{
						Method:  httpMethod,
						Route:   routing.ToEndpoint(route),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         exceptionFilterHandler.exceptionFilterable.Catch,
							Name:            exceptionFilterableType.String(),
							MainHandlerName: r.PatternToFnNameMap[pattern],
						},
					},
				})
			}
		}
	}

	return exceptionFilterItemArr
}

func (e *ExceptionFilter) InjectProvidersIntoWSExceptionFilters(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
	exceptionFilterItemArr := make([]ExceptionFilterItem, 0, len(ws.patternToFnNameMap)*len(e.ExceptionFilterHandlers))

	for _, exceptionFilterHandler := range e.ExceptionFilterHandlers {
		exceptionFilterableType := reflect.TypeOf(exceptionFilterHandler.exceptionFilterable)
		exceptionFilterableValue := reflect.ValueOf(exceptionFilterHandler.exceptionFilterable)
		newExceptionFilter := reflect.New(exceptionFilterableType)

		for i := 0; i < exceptionFilterableType.NumField(); i++ {
			cb(i, exceptionFilterableType, exceptionFilterableValue, newExceptionFilter)
		}

		newExceptionFilterable := newExceptionFilter.Interface()
		newExceptionFilterable = Construct(newExceptionFilterable, "NewExceptionFilter")
		exceptionFilterHandler.exceptionFilterable = newExceptionFilterable.(ExceptionFilterable)

		shouldAddExceptionFilter := map[string]bool{}
		for _, handler := range exceptionFilterHandler.handlers {
			fnName := GetFnName(handler)
			_, eventName, _ := ParseFnNameToURL(fnName, WSOperations)
			eventName = ToWSEventName(eventName)
			shouldAddExceptionFilter[eventName] = true
		}
		applyAll := len(shouldAddExceptionFilter) == 0

		for pattern := range ws.patternToFnNameMap {
			if applyAll || shouldAddExceptionFilter[pattern] {
				exceptionFilterItemArr = append(exceptionFilterItemArr, ExceptionFilterItem{
					WS: WSExceptionFilterItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: exceptionFilterHandler.exceptionFilterable.Catch,
							Name:    exceptionFilterableType.String(),
						},
					},
				})
			}
		}
	}

	return exceptionFilterItemArr
}
