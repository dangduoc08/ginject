package common

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/routing"
)

type HTTPCatch = func(*ctx.HTTPContext, *exception.Exception)

type HTTPExceptionFilterItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
	Common  CommonItem
}

func AsHTTPExceptionFilter(exceptionFilterable any) (HTTPCatch, bool) {
	method := reflect.ValueOf(exceptionFilterable).MethodByName(ExceptionFilterMethodName)
	if !method.IsValid() {
		return nil, false
	}
	fn, ok := method.Interface().(HTTPCatch)
	return fn, ok
}

func (e *ExceptionFilter) InjectProvidersIntoHTTPExceptionFilters(r *HTTP, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
	exceptionFilterItemArr := make([]ExceptionFilterItem, 0, len(r.PatternToFuncNameMap)*len(e.ExceptionFilterHandlers))

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

		catch, ok := AsHTTPExceptionFilter(exceptionFilterHandler.exceptionFilterable)
		if !ok {
			if _, ok = AsWSExceptionFilter(exceptionFilterHandler.exceptionFilterable); ok {
				continue
			}

			panic(errors.New(color.FmtRed(
				"invalid exception filter: %v.%s must be func(*ctx.HTTPContext, *exception.Exception) to be bound as a HTTP exception filter",
				exceptionFilterableType,
				ExceptionFilterMethodName,
			)))
		}

		targetedPatterns := map[string]bool{}
		for _, handler := range exceptionFilterHandler.handlers {
			fnName := GetFuncName(handler)
			if pattern, ok := r.FuncNameToPatternMap[fnName]; ok {
				targetedPatterns[pattern] = true
			}
		}
		applyAll := len(targetedPatterns) == 0

		for pattern := range r.PatternToFuncNameMap {
			if applyAll || targetedPatterns[pattern] {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				exceptionFilterItemArr = append(exceptionFilterItemArr, ExceptionFilterItem{
					HTTP: HTTPExceptionFilterItem{
						Method:  httpMethod,
						Route:   str.Enclose(route, '/'),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         catch,
							Name:            exceptionFilterableType.String(),
							MainHandlerName: r.PatternToFuncNameMap[pattern],
						},
					},
				})
			}
		}
	}

	return exceptionFilterItemArr
}

func RunHTTPCatchChain(c *ctx.HTTPContext, catchFns []HTTPCatch, rec any) {
	if len(catchFns) == 0 {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			RunHTTPCatchChain(c, catchFns[1:], r)
		}
	}()

	catchFns[0](c, NormalizeRecovered(rec))
}
