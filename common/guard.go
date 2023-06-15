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

type GuardItem struct {
	Method  string
	Route   string
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

func (g *Guard) AddGuardsToModule(r *Rest, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
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

		for pattern := range r.patternToFnNameMap {
			httpMethod, route := routing.SplitRoute(pattern)
			guardItemArr = append(guardItemArr, GuardItem{
				Method:  httpMethod,
				Route:   routing.ToEndpoint(route),
				Handler: guardHandler.Guarder.CanActivate,
			})
		}
	}

	return guardItemArr
}
