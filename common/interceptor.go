package common

import (
	"reflect"

	"github.com/dangduoc08/gogo/aggregation"
	"github.com/dangduoc08/gogo/ctx"
	"github.com/dangduoc08/gogo/routing"
)

type Intercept = func(*ctx.Context, *aggregation.Aggregation) any

type Interceptable interface {
	Intercept(*ctx.Context, *aggregation.Aggregation) any
}

type InterceptorHandler struct {
	Interceptable Interceptable
	Handlers      []any
}

type InterceptorItem struct {
	// for REST
	Method  string
	Route   string
	Version string

	// for WS
	EventName string

	Handler any
}

type Interceptor struct {
	InterceptorHandlers []InterceptorHandler
}

func (i *Interceptor) BindInterceptor(interceptable Interceptable, handlers ...any) *Interceptor {
	interceptorHandler := InterceptorHandler{
		Interceptable: interceptable,
		Handlers:      handlers,
	}

	i.InterceptorHandlers = append(i.InterceptorHandlers, interceptorHandler)

	return i
}

func (i *Interceptor) InjectProvidersIntoRESTInterceptors(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []InterceptorItem {
	interceptorItemArr := []InterceptorItem{}

	for _, interceptorHandler := range i.InterceptorHandlers {

		interceptableType := reflect.TypeOf(interceptorHandler.Interceptable)
		interceptableValue := reflect.ValueOf(interceptorHandler.Interceptable)
		newInterceptor := reflect.New(interceptableType)

		for i := 0; i < interceptableType.NumField(); i++ {

			// callback use to inject providers
			cb(i, interceptableType, interceptableValue, newInterceptor)
		}

		// invoke interceptor constructor
		// if NewInterceptor was declared
		newInterceptable := newInterceptor.Interface()
		newInterceptable = Construct(newInterceptable, "NewInterceptor")

		interceptorHandler.Interceptable = newInterceptable.(Interceptable)

		shouldAddInterceptors := map[string]bool{}
		for _, handler := range interceptorHandler.Handlers {
			fnName := GetFnName(handler)
			method, route, version := ParseFnNameToURL(fnName, RESTOperations)
			httpMethod := routing.OperationsMapHTTPMethods[method]

			route = r.addPrefixesToRoute(route, fnName, r.GetPrefixes())
			shouldAddInterceptors[routing.MethodRouteVersionToPattern(httpMethod, route, version)] = true
		}

		for pattern := range r.PatternToFnNameMap {
			if _, ok := shouldAddInterceptors[pattern]; ok || len(shouldAddInterceptors) == 0 {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				interceptorItemArr = append(interceptorItemArr, InterceptorItem{
					Method:  httpMethod,
					Route:   routing.ToEndpoint(route),
					Version: version,
					Handler: interceptorHandler.Interceptable.Intercept,
				})
			}
		}
	}

	return interceptorItemArr
}

func (i *Interceptor) InjectProvidersIntoWSInterceptors(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []InterceptorItem {
	interceptorItemArr := []InterceptorItem{}

	for _, interceptorHandler := range i.InterceptorHandlers {

		interceptableType := reflect.TypeOf(interceptorHandler.Interceptable)
		interceptableValue := reflect.ValueOf(interceptorHandler.Interceptable)
		newInterceptor := reflect.New(interceptableType)

		for i := 0; i < interceptableType.NumField(); i++ {

			// callback use to inject providers
			cb(i, interceptableType, interceptableValue, newInterceptor)
		}

		// invoke interceptor constructor
		// if NewInterceptor was declared
		newInterceptable := newInterceptor.Interface()
		newInterceptable = Construct(newInterceptable, "NewInterceptor")

		interceptorHandler.Interceptable = newInterceptable.(Interceptable)

		shouldAddInterceptors := map[string]bool{}
		for _, handler := range interceptorHandler.Handlers {
			fnName := GetFnName(handler)
			_, eventName, _ := ParseFnNameToURL(fnName, WSOperations)
			eventName = ToWSEventName(ws.subprotocol, eventName)
			shouldAddInterceptors[eventName] = true
		}

		for pattern := range ws.patternToFnNameMap {
			if _, ok := shouldAddInterceptors[pattern]; ok || len(shouldAddInterceptors) == 0 {
				interceptorItemArr = append(interceptorItemArr, InterceptorItem{
					EventName: pattern,
					Handler:   interceptorHandler.Interceptable.Intercept,
				})
			}
		}
	}

	return interceptorItemArr
}
