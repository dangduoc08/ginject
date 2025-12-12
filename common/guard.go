package common

import (
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/routing"
)

type CanActivate = func(*ctx.Context) bool

type Guarder interface {
	CanActivate(*ctx.Context) bool
}

type RESTGuardItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
	Common  CommonItem
}

type WSGuardItem struct {
	EventName string
	Common    CommonItem
}

type GuardItem struct {
	REST RESTGuardItem
	WS   WSGuardItem
}

type guardHandler struct {
	guarder  Guarder
	handlers []any
}

type Guard struct {
	GuardHandlers []guardHandler
}

func (g *Guard) BindGuard(guarder Guarder, handlers ...any) *Guard {
	guardHandler := guardHandler{
		guarder:  guarder,
		handlers: handlers,
	}

	g.GuardHandlers = append(g.GuardHandlers, guardHandler)
	return g
}

func (g *Guard) InjectProvidersIntoRESTGuards(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
	guardItemArr := []GuardItem{}

	for _, guardHandler := range g.GuardHandlers {
		guarderType := reflect.TypeOf(guardHandler.guarder)
		guarderValue := reflect.ValueOf(guardHandler.guarder)
		newGuard := reflect.New(guarderType)

		for i := 0; i < guarderType.NumField(); i++ {

			// callback use to inject providers
			cb(i, guarderType, guarderValue, newGuard)
		}

		// invoke guard constructor
		// if NewGuard was declared
		newGuarder := newGuard.Interface()
		newGuarder = Construct(newGuarder, "NewGuard")

		guardHandler.guarder = newGuarder.(Guarder)

		shouldAddGuard := map[string]bool{}
		for _, handler := range guardHandler.handlers {
			fnName := GetFnName(handler)
			if pattern, ok := r.FnNameToPatternMap[fnName]; ok {
				shouldAddGuard[pattern] = true
			}
		}

		for pattern := range r.PatternToFnNameMap {
			if _, ok := shouldAddGuard[pattern]; ok || len(shouldAddGuard) == 0 {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				guardItemArr = append(guardItemArr, GuardItem{
					REST: RESTGuardItem{
						Method:  httpMethod,
						Route:   routing.ToEndpoint(route),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         guardHandler.guarder.CanActivate,
							Name:            guarderType.String(),
							MainHandlerName: r.PatternToFnNameMap[pattern],
						},
					},
				})
			}
		}
	}

	return guardItemArr
}

func (g *Guard) InjectProvidersIntoWSGuards(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
	guardItemArr := []GuardItem{}

	for _, guardHandler := range g.GuardHandlers {
		guarderType := reflect.TypeOf(guardHandler.guarder)
		guarderValue := reflect.ValueOf(guardHandler.guarder)
		newGuard := reflect.New(guarderType)

		for i := 0; i < guarderType.NumField(); i++ {

			// callback use to inject providers
			cb(i, guarderType, guarderValue, newGuard)
		}

		// invoke guard constructor
		// if NewGuard was declared
		newGuarder := newGuard.Interface()
		newGuarder = Construct(newGuarder, "NewGuard")

		guardHandler.guarder = newGuarder.(Guarder)

		shouldAddGuard := map[string]bool{}
		for _, handler := range guardHandler.handlers {
			fnName := GetFnName(handler)
			_, eventName, _ := ParseFnNameToURL(fnName, WSOperations)
			eventName = ToWSEventName(ws.subprotocol, eventName)
			shouldAddGuard[eventName] = true
		}

		for pattern := range ws.patternToFnNameMap {
			if _, ok := shouldAddGuard[pattern]; ok || len(shouldAddGuard) == 0 {
				guardItemArr = append(guardItemArr, GuardItem{
					WS: WSGuardItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: guardHandler.guarder.CanActivate,
							Name:    guarderType.String(),
						},
					},
				})
			}
		}
	}

	return guardItemArr
}
