package common

import (
	"context"
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/routing"
)

type HTTPIntercept = func(*ctx.HTTPContext, *aggregation.Aggregation) any

type HTTPInterceptorItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
	Common  CommonItem
}

func AsHTTPInterceptor(interceptable any) (HTTPIntercept, bool) {
	method := reflect.ValueOf(interceptable).MethodByName(InterceptorMethodName)
	if !method.IsValid() {
		return nil, false
	}
	fn, ok := method.Interface().(HTTPIntercept)
	return fn, ok
}

func (i *Interceptor) InjectProvidersIntoHTTPInterceptors(r *HTTP, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []InterceptorItem {
	interceptorItemArr := make([]InterceptorItem, 0, len(r.PatternToFuncNameMap)*len(i.InterceptorHandlers))

	for _, interceptorHandler := range i.InterceptorHandlers {
		interceptableType := reflect.TypeOf(interceptorHandler.interceptable)
		interceptableValue := reflect.ValueOf(interceptorHandler.interceptable)
		newInterceptor := reflect.New(interceptableType)

		for j := 0; j < interceptableType.NumField(); j++ {
			cb(j, interceptableType, interceptableValue, newInterceptor)
		}

		newInterceptable := newInterceptor.Interface()
		newInterceptable = Construct(newInterceptable, "NewInterceptor")
		interceptorHandler.interceptable = newInterceptable.(Interceptable)

		intercept, ok := AsHTTPInterceptor(interceptorHandler.interceptable)
		if !ok {
			if _, ok = AsWSInterceptor(interceptorHandler.interceptable); ok {
				continue
			}

			panic(errors.New(color.FmtRed(
				"invalid interceptor: %v.%s must be func(*ctx.HTTPContext, *aggregation.Aggregation) any to be bound as a HTTP interceptor",
				interceptableType,
				InterceptorMethodName,
			)))
		}

		targetedPatterns := map[string]bool{}
		for _, handler := range interceptorHandler.handlers {
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

				interceptorItemArr = append(interceptorItemArr, InterceptorItem{
					HTTP: HTTPInterceptorItem{
						Method:  httpMethod,
						Route:   str.Enclose(route, '/'),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         intercept,
							Name:            interceptableType.String(),
							MainHandlerName: r.PatternToFuncNameMap[pattern],
						},
					},
				})
			}
		}
	}

	return interceptorItemArr
}

func BuildHTTPInterceptMiddleware(key string, interceptFn HTTPIntercept) ctx.HTTPHandler {
	return func(c *ctx.HTTPContext) {
		aggregationInstance := aggregation.NewAggregation()

		if aggregations, ok := c.Context().Value(WithValueKey(key)).([]*aggregation.Aggregation); ok {
			aggregations = append(aggregations, aggregationInstance)
			c.Request = c.WithContext(context.WithValue(c.Context(), WithValueKey(key), aggregations))
		} else {
			c.Request = c.WithContext(context.WithValue(c.Context(), WithValueKey(key), []*aggregation.Aggregation{aggregationInstance}))
		}

		aggregationInstance.IsMainHandlerCalled = false
		aggregationInstance.SetMainData(nil)

		aggregationInstance.InterceptorData = interceptFn(c, aggregationInstance)

		c.Next()
	}
}
