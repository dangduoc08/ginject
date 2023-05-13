package common

import (
	"reflect"

	"github.com/dangduoc08/gooh/context"
	"github.com/dangduoc08/gooh/routing"
)

type Guarder interface {
	CanActivate(*context.Context) bool
}

type GuardHandler struct {
	Guarder  Guarder
	Handlers []any
}

type Guard struct {
	GuardHandlers []GuardHandler
}

func (g *Guard) addGuardToRoute(httpMethod, route string, router *routing.Router, guarder Guarder) {
	router.For(route, []string{httpMethod})(func(ctx *context.Context) {
		HandleGuard(ctx, guarder.CanActivate(ctx))
	})
}

func (g *Guard) BindGuard(guarder Guarder, handlers ...any) *Guard {
	guardHandler := GuardHandler{
		Guarder:  guarder,
		Handlers: handlers,
	}

	g.GuardHandlers = append(g.GuardHandlers, guardHandler)
	return g
}

func (g *Guard) AddGuardsToController(r *Rest, router *routing.Router, cb func(int, reflect.Type, reflect.Value, reflect.Value)) {
	for _, guardHandler := range g.GuardHandlers {

		guarderType := reflect.TypeOf(guardHandler.Guarder)
		guarderValue := reflect.ValueOf(guardHandler.Guarder)
		newGuard := reflect.New(guarderType)

		for i := 0; i < guarderType.NumField(); i++ {
			cb(i, guarderType, guarderValue, newGuard)
		}

		// invoke guard constructor
		// if NewGuard was declared
		newGuarder := newGuard.Interface().(Guarder)
		newGuarderValue := reflect.ValueOf(newGuarder)
		guardConstructor := newGuarderValue.MethodByName("NewGuard")
		if guardConstructor.IsValid() {
			newGuarder = guardConstructor.Call([]reflect.Value{})[0].Interface().(Guarder)
		}

		guardHandler.Guarder = newGuarder
		for pattern, fnName := range r.patternToFnNameMap {
			httpMethod, route := routing.SplitRoute(pattern)

			// Guard all methods
			if len(guardHandler.Handlers) == 0 {
				g.addGuardToRoute(httpMethod, route, router, guardHandler.Guarder)
			} else {
				for _, handler := range guardHandler.Handlers {
					parsedFnName := getFnName(handler)
					if parsedFnName == fnName {
						httpMethod, route := routing.SplitRoute(pattern)
						g.addGuardToRoute(httpMethod, route, router, guardHandler.Guarder)
					}
				}
			}
		}
	}
}
