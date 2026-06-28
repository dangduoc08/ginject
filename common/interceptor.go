package common

import (
	"reflect"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/routing"
)

type Intercept = func(*ctx.Context, *aggregation.Aggregation) any

type Interceptable interface {
	Intercept(*ctx.Context, *aggregation.Aggregation) any
}

type RESTInterceptorItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
	Common  CommonItem
}

type WSInterceptorItem struct {
	EventName string
	Common    CommonItem
}

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

func (i *Interceptor) InjectProvidersIntoRESTInterceptors(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []InterceptorItem {
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

		shouldAddInterceptors := map[string]bool{}
		for _, handler := range interceptorHandler.handlers {
			fnName := GetFuncName(handler)
			if pattern, ok := r.FuncNameToPatternMap[fnName]; ok {
				shouldAddInterceptors[pattern] = true
			}
		}
		applyAll := len(shouldAddInterceptors) == 0

		for pattern := range r.PatternToFuncNameMap {
			if applyAll || shouldAddInterceptors[pattern] {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				interceptorItemArr = append(interceptorItemArr, InterceptorItem{
					REST: RESTInterceptorItem{
						Method:  httpMethod,
						Route:   str.Enclose(route, '/'),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         interceptorHandler.interceptable.Intercept,
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

func (i *Interceptor) InjectProvidersIntoWSInterceptors(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []InterceptorItem {
	interceptorItemArr := make([]InterceptorItem, 0, len(ws.patternToFuncNameMap)*len(i.InterceptorHandlers))

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

		shouldAddInterceptors := map[string]bool{}
		for _, handler := range interceptorHandler.handlers {
			fnName := GetFuncName(handler)
			if event, ok := ParseWSFuncNameToEvent(fnName); ok {
				shouldAddInterceptors[event] = true
			}
		}
		applyAll := len(shouldAddInterceptors) == 0

		for pattern := range ws.patternToFuncNameMap {
			if applyAll || shouldAddInterceptors[pattern] {
				interceptorItemArr = append(interceptorItemArr, InterceptorItem{
					WS: WSInterceptorItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: interceptorHandler.interceptable.Intercept,
							Name:    interceptableType.String(),
						},
					},
				})
			}
		}
	}

	return interceptorItemArr
}
