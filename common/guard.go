package common

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/internal/color"
)

const GuardMethodName = "CanActivate"

type Guarder any

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

func GuardShapeError(guarder any) error {
	return errors.New(color.FmtRed(
		"invalid handler: %v has no %s method usable as a guard",
		reflect.TypeOf(guarder),
		GuardMethodName,
	))
}
