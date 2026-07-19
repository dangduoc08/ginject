package common

import (
	"context"
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/color"
)

type WSIntercept = func(*ctx.WSContext, *aggregation.Aggregation) any

type WSInterceptorItem struct {
	EventName string
	Common    CommonItem
}

func AsWSInterceptor(interceptable any) (WSIntercept, bool) {
	method := reflect.ValueOf(interceptable).MethodByName(InterceptorMethodName)
	if !method.IsValid() {
		return nil, false
	}
	fn, ok := method.Interface().(WSIntercept)
	return fn, ok
}

func (i *Interceptor) InjectProvidersIntoWSInterceptors(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []InterceptorItem {
	interceptorItemArr := make([]InterceptorItem, 0, len(ws.funcNameByEvent)*len(i.InterceptorHandlers))

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

		intercept, ok := AsWSInterceptor(interceptorHandler.interceptable)
		if !ok {
			panic(errors.New(color.FmtRed(
				"invalid handler: %v.%s must be func(*ctx.WSContext, *aggregation.Aggregation) any to be bound as a WS interceptor",
				interceptableType,
				InterceptorMethodName,
			)))
		}

		targetedPatterns := map[string]bool{}
		for _, handler := range interceptorHandler.handlers {
			fnName := GetFuncName(handler)
			if event, ok := ParseWSFuncNameToEvent(fnName); ok {
				targetedPatterns[event] = true
			}
		}
		applyAll := len(targetedPatterns) == 0

		for pattern := range ws.funcNameByEvent {
			if applyAll || targetedPatterns[pattern] {
				interceptorItemArr = append(interceptorItemArr, InterceptorItem{
					WS: WSInterceptorItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: intercept,
							Name:    interceptableType.String(),
						},
					},
				})
			}
		}
	}

	return interceptorItemArr
}

func BuildWSInterceptMiddleware(key string, interceptFn WSIntercept) ctx.WSHandler {
	return func(c *ctx.WSContext) {
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
