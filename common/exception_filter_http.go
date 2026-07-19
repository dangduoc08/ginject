package common

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/broker"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/routing"
)

type RESTCatch = func(*ctx.HTTPContext, *exception.Exception)

type RESTExceptionFilterItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
	Common  CommonItem
}

func AsRESTExceptionFilter(exceptionFilterable any) (RESTCatch, bool) {
	method := reflect.ValueOf(exceptionFilterable).MethodByName(ExceptionFilterMethodName)
	if !method.IsValid() {
		return nil, false
	}
	fn, ok := method.Interface().(RESTCatch)
	return fn, ok
}

func (e *ExceptionFilter) InjectProvidersIntoRESTExceptionFilters(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
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

		catch, ok := AsRESTExceptionFilter(exceptionFilterHandler.exceptionFilterable)
		if !ok {
			panic(errors.New(color.FmtRed(
				"invalid handler: %v.%s must be func(*ctx.HTTPContext, *exception.Exception) to be bound as a REST exception filter",
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
					REST: RESTExceptionFilterItem{
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

func BuildHTTPCatchMiddleware(catchEvent string, catchFns []RESTCatch) ctx.HTTPHandler {
	return func(c *ctx.HTTPContext) {
		_, _ = c.Broker.Subscribe(catchEvent, func(m *broker.Message) {
			p := m.Payload.(CatchEventPayload)
			catchFnIndex := p.Index

			defer func() {
				if rec := recover(); rec != nil {
					_ = c.Broker.Publish(catchEvent, CatchEventPayload{ReqCtx: p.ReqCtx, Recovered: rec, Index: catchFnIndex + 1})
				}
			}()

			catchFns[catchFnIndex](p.ReqCtx, NormalizeRecovered(p.Recovered))
		})

		c.Next()
	}
}
