package common

import (
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/routing"
)

type Use = func(*ctx.Context, ctx.Next)

type MiddlewareFn interface {
	Use(*ctx.Context, ctx.Next)
}

type RESTMiddlewareItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
	Common  CommonItem
}

type WSMiddlewareItem struct {
	EventName string
	Common    CommonItem
}

type MiddlewareItem struct {
	REST RESTMiddlewareItem
	WS   WSMiddlewareItem
}

type middlewareHandler struct {
	middlewareFn MiddlewareFn
	handlers     []any
}

type Middleware struct {
	MiddlewareHandlers []middlewareHandler
}

func (m *Middleware) BindMiddleware(middlewareFn MiddlewareFn, handlers ...any) *Middleware {
	middlewareHandler := middlewareHandler{
		middlewareFn: middlewareFn,
		handlers:     handlers,
	}

	m.MiddlewareHandlers = append(m.MiddlewareHandlers, middlewareHandler)
	return m
}

func (m *Middleware) InjectProvidersIntoRESTMiddlewares(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []MiddlewareItem {
	middlewareItemArr := make([]MiddlewareItem, 0, len(r.PatternToFnNameMap)*len(m.MiddlewareHandlers))

	for _, middlewareHandler := range m.MiddlewareHandlers {
		middlewarerType := reflect.TypeOf(middlewareHandler.middlewareFn)
		middlewarerValue := reflect.ValueOf(middlewareHandler.middlewareFn)
		newMiddleware := reflect.New(middlewarerType)

		for i := 0; i < middlewarerType.NumField(); i++ {
			cb(i, middlewarerType, middlewarerValue, newMiddleware)
		}

		newMiddlewareFn := newMiddleware.Interface()
		newMiddlewareFn = Construct(newMiddlewareFn, "NewMiddleware")
		middlewareHandler.middlewareFn = newMiddlewareFn.(MiddlewareFn)

		shouldAddMiddleware := map[string]bool{}
		for _, handler := range middlewareHandler.handlers {
			fnName := GetFnName(handler)
			if pattern, ok := r.FnNameToPatternMap[fnName]; ok {
				shouldAddMiddleware[pattern] = true
			}
		}
		applyAll := len(shouldAddMiddleware) == 0

		for pattern := range r.PatternToFnNameMap {
			if applyAll || shouldAddMiddleware[pattern] {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				middlewareItemArr = append(middlewareItemArr, MiddlewareItem{
					REST: RESTMiddlewareItem{
						Method:  httpMethod,
						Route:   routing.ToEndpoint(route),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         middlewareHandler.middlewareFn.Use,
							Name:            middlewarerType.String(),
							MainHandlerName: r.PatternToFnNameMap[pattern],
						},
					},
				})
			}
		}
	}

	return middlewareItemArr
}

func (g *Middleware) InjectProvidersIntoWSMiddlewares(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []MiddlewareItem {
	middlewareItemArr := make([]MiddlewareItem, 0, len(ws.patternToFnNameMap)*len(g.MiddlewareHandlers))

	for _, middlewareHandler := range g.MiddlewareHandlers {
		middlewarerType := reflect.TypeOf(middlewareHandler.middlewareFn)
		middlewarerValue := reflect.ValueOf(middlewareHandler.middlewareFn)
		newMiddleware := reflect.New(middlewarerType)

		for i := 0; i < middlewarerType.NumField(); i++ {
			cb(i, middlewarerType, middlewarerValue, newMiddleware)
		}

		newMiddlewarer := newMiddleware.Interface()
		newMiddlewarer = Construct(newMiddlewarer, "NewMiddleware")
		middlewareHandler.middlewareFn = newMiddlewarer.(MiddlewareFn)

		shouldAddMiddleware := map[string]bool{}
		for _, handler := range middlewareHandler.handlers {
			fnName := GetFnName(handler)
			_, eventName, _ := ParseFnNameToURL(fnName, WSOperations)
			eventName = ToWSEventName(eventName)
			shouldAddMiddleware[eventName] = true
		}
		applyAll := len(shouldAddMiddleware) == 0

		for pattern := range ws.patternToFnNameMap {
			if applyAll || shouldAddMiddleware[pattern] {
				middlewareItemArr = append(middlewareItemArr, MiddlewareItem{
					WS: WSMiddlewareItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: middlewareHandler.middlewareFn.Use,
							Name:    middlewarerType.String(),
						},
					},
				})
			}
		}
	}

	return middlewareItemArr
}
