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
	guardItemArr := make([]GuardItem, 0, len(r.PatternToFuncNameMap)*len(g.GuardHandlers))

	for _, guardHandler := range g.GuardHandlers {
		guarderType := reflect.TypeOf(guardHandler.guarder)
		guarderValue := reflect.ValueOf(guardHandler.guarder)
		newGuard := reflect.New(guarderType)

		for i := 0; i < guarderType.NumField(); i++ {
			cb(i, guarderType, guarderValue, newGuard)
		}

		newGuarder := newGuard.Interface()
		newGuarder = Construct(newGuarder, "NewGuard")
		guardHandler.guarder = newGuarder.(Guarder)

		shouldAddGuard := map[string]bool{}
		for _, handler := range guardHandler.handlers {
			fnName := GetFuncName(handler)
			if pattern, ok := r.FuncNameToPatternMap[fnName]; ok {
				shouldAddGuard[pattern] = true
			}
		}
		applyAll := len(shouldAddGuard) == 0

		for pattern := range r.PatternToFuncNameMap {
			if applyAll || shouldAddGuard[pattern] {
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
							MainHandlerName: r.PatternToFuncNameMap[pattern],
						},
					},
				})
			}
		}
	}

	return guardItemArr
}

func (g *Guard) InjectProvidersIntoWSGuards(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
	guardItemArr := make([]GuardItem, 0, len(ws.patternToFuncNameMap)*len(g.GuardHandlers))

	for _, guardHandler := range g.GuardHandlers {
		guarderType := reflect.TypeOf(guardHandler.guarder)
		guarderValue := reflect.ValueOf(guardHandler.guarder)
		newGuard := reflect.New(guarderType)

		for i := 0; i < guarderType.NumField(); i++ {
			cb(i, guarderType, guarderValue, newGuard)
		}

		newGuarder := newGuard.Interface()
		newGuarder = Construct(newGuarder, "NewGuard")
		guardHandler.guarder = newGuarder.(Guarder)

		shouldAddGuard := map[string]bool{}
		for _, handler := range guardHandler.handlers {
			fnName := GetFuncName(handler)
			if event, ok := ParseWSFuncNameToEvent(fnName); ok {
				shouldAddGuard[event] = true
			}
		}
		applyAll := len(shouldAddGuard) == 0

		for pattern := range ws.patternToFuncNameMap {
			if applyAll || shouldAddGuard[pattern] {
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
