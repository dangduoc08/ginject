package common

import (
	"reflect"

	"github.com/dangduoc08/ginject/aggregation"
	"github.com/dangduoc08/ginject/ctx"
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
	interceptorItemArr := []InterceptorItem{}

	for _, interceptorHandler := range i.InterceptorHandlers {

		interceptableType := reflect.TypeOf(interceptorHandler.interceptable)
		interceptableValue := reflect.ValueOf(interceptorHandler.interceptable)
		newInterceptor := reflect.New(interceptableType)

		for i := 0; i < interceptableType.NumField(); i++ {

			// callback use to inject providers
			cb(i, interceptableType, interceptableValue, newInterceptor)
		}

		// invoke interceptor constructor
		// if NewInterceptor was declared
		newInterceptable := newInterceptor.Interface()
		newInterceptable = Construct(newInterceptable, "NewInterceptor")

		interceptorHandler.interceptable = newInterceptable.(Interceptable)

		shouldAddInterceptors := map[string]bool{}
		for _, handler := range interceptorHandler.handlers {
			fnName := GetFnName(handler)
			if pattern, ok := r.FnNameToPatternMap[fnName]; ok {
				shouldAddInterceptors[pattern] = true
			}
		}

		for pattern := range r.PatternToFnNameMap {
			if _, ok := shouldAddInterceptors[pattern]; ok || len(shouldAddInterceptors) == 0 {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				interceptorItemArr = append(interceptorItemArr, InterceptorItem{
					REST: RESTInterceptorItem{
						Method:  httpMethod,
						Route:   routing.ToEndpoint(route),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         interceptorHandler.interceptable.Intercept,
							Name:            interceptableType.String(),
							MainHandlerName: r.PatternToFnNameMap[pattern],
						},
					},
				})
			}
		}
	}

	return interceptorItemArr
}

func (i *Interceptor) InjectProvidersIntoWSInterceptors(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []InterceptorItem {
	interceptorItemArr := []InterceptorItem{}

	for _, interceptorHandler := range i.InterceptorHandlers {

		interceptableType := reflect.TypeOf(interceptorHandler.interceptable)
		interceptableValue := reflect.ValueOf(interceptorHandler.interceptable)
		newInterceptor := reflect.New(interceptableType)

		for i := 0; i < interceptableType.NumField(); i++ {

			// callback use to inject providers
			cb(i, interceptableType, interceptableValue, newInterceptor)
		}

		// invoke interceptor constructor
		// if NewInterceptor was declared
		newInterceptable := newInterceptor.Interface()
		newInterceptable = Construct(newInterceptable, "NewInterceptor")

		interceptorHandler.interceptable = newInterceptable.(Interceptable)

		shouldAddInterceptors := map[string]bool{}
		for _, handler := range interceptorHandler.handlers {
			fnName := GetFnName(handler)
			_, eventName, _ := ParseFnNameToURL(fnName, WSOperations)
			eventName = ToWSEventName(ws.subprotocol, eventName)
			shouldAddInterceptors[eventName] = true
		}

		for pattern := range ws.patternToFnNameMap {
			if _, ok := shouldAddInterceptors[pattern]; ok || len(shouldAddInterceptors) == 0 {
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
