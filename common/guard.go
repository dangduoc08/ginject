package common

import (
	"reflect"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
)

type CanActivate = func(*context.Context) bool

type Guarder interface {
	CanActivate(*context.Context) bool
}

type GuardHandler struct {
	Guarder  Guarder
	Handlers []any
}

type GuardItem struct {

	// for REST
	Method string
	Route  string

	// for WS
	EventName string

	Handler any
}

type Guard struct {
	GuardHandlers []GuardHandler
}

func (g *Guard) BindGuard(guarder Guarder, handlers ...any) *Guard {
	guardHandler := GuardHandler{
		Guarder:  guarder,
		Handlers: handlers,
	}

	g.GuardHandlers = append(g.GuardHandlers, guardHandler)
	return g
}

func (g *Guard) InjectProvidersIntoRESTGuards(r *REST, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
	guardItemArr := []GuardItem{}

	for _, guardHandler := range g.GuardHandlers {
		guarderType := reflect.TypeOf(guardHandler.Guarder)
		guarderValue := reflect.ValueOf(guardHandler.Guarder)
		newGuard := reflect.New(guarderType)

		for i := 0; i < guarderType.NumField(); i++ {

			// callback use to inject providers
			cb(i, guarderType, guarderValue, newGuard)
		}

		// invoke guard constructor
		// if NewGuard was declared
		newGuarder := newGuard.Interface()
		newGuarder = Construct(newGuarder, "NewGuard")

		guardHandler.Guarder = newGuarder.(Guarder)

		shouldAddGuard := map[string]bool{}
		for _, handler := range guardHandler.Handlers {
			fnName := GetFnName(handler)
			httpMethod, route := ParseFnNameToURL(fnName, RESTOperations)
			route = r.addPrefixesToRoute(route, fnName, r.GetPrefixes())
			shouldAddGuard[routing.AddMethodToRoute(route, httpMethod)] = true
		}

		for pattern := range r.patternToFnNameMap {
			if _, ok := shouldAddGuard[pattern]; ok || len(shouldAddGuard) == 0 {
				httpMethod, route := routing.SplitRoute(pattern)
				guardItemArr = append(guardItemArr, GuardItem{
					Method:  httpMethod,
					Route:   routing.ToEndpoint(route),
					Handler: guardHandler.Guarder.CanActivate,
				})
			}
		}
	}

	return guardItemArr
}

func (g *Guard) InjectProvidersIntoWSGuards(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
	guardItemArr := []GuardItem{}

	for _, guardHandler := range g.GuardHandlers {
		guarderType := reflect.TypeOf(guardHandler.Guarder)
		guarderValue := reflect.ValueOf(guardHandler.Guarder)
		newGuard := reflect.New(guarderType)

		for i := 0; i < guarderType.NumField(); i++ {

			// callback use to inject providers
			cb(i, guarderType, guarderValue, newGuard)
		}

		// invoke guard constructor
		// if NewGuard was declared
		newGuarder := newGuard.Interface()
		newGuarder = Construct(newGuarder, "NewGuard")

		guardHandler.Guarder = newGuarder.(Guarder)

		shouldAddGuard := map[string]bool{}
		for _, handler := range guardHandler.Handlers {
			fnName := GetFnName(handler)
			_, eventName := ParseFnNameToURL(fnName, WSOperations)
			eventName = ToWSEventName(ws.subprotocol, eventName)
			shouldAddGuard[eventName] = true
		}

		for pattern := range ws.patternToFnNameMap {
			if _, ok := shouldAddGuard[pattern]; ok || len(shouldAddGuard) == 0 {
				guardItemArr = append(guardItemArr, GuardItem{
					EventName: pattern,
					Handler:   guardHandler.Guarder.CanActivate,
				})
			}
		}
	}

	return guardItemArr
}
