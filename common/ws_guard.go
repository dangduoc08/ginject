package common

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/color"
)

type WSCanActivate = func(*ctx.WSContext) bool

type WSGuardItem struct {
	EventName string
	Common    CommonItem
}

func AsWSGuard(guarder any) (WSCanActivate, bool) {
	method := reflect.ValueOf(guarder).MethodByName(GuardMethodName)
	if !method.IsValid() {
		return nil, false
	}
	fn, ok := method.Interface().(WSCanActivate)
	return fn, ok
}

func (g *Guard) InjectProvidersIntoWSGuards(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
	guardItemArr := make([]GuardItem, 0, len(ws.funcNameByEvent)*len(g.GuardHandlers))

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

		canActivate, ok := AsWSGuard(guardHandler.guarder)
		if !ok {
			if _, ok = AsRESTGuard(guardHandler.guarder); ok {
				continue
			}

			panic(errors.New(color.FmtRed(
				"invalid guard: %v.%s must be func(*ctx.WSContext) bool to be bound as a WS guard",
				guarderType,
				GuardMethodName,
			)))
		}

		targetedPatterns := map[string]bool{}
		for _, handler := range guardHandler.handlers {
			fnName := GetFuncName(handler)
			if event, ok := ParseWSFuncNameToEvent(fnName); ok {
				targetedPatterns[event] = true
			}
		}
		applyAll := len(targetedPatterns) == 0

		for pattern := range ws.funcNameByEvent {
			if applyAll || targetedPatterns[pattern] {
				guardItemArr = append(guardItemArr, GuardItem{
					WS: WSGuardItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: canActivate,
							Name:    guarderType.String(),
						},
					},
				})
			}
		}
	}

	return guardItemArr
}

func BuildWSGuardMiddleware(canActivateFn WSCanActivate) ctx.WSHandler {
	return func(c *ctx.WSContext) { handleWSGuard(c, canActivateFn(c)) }
}

func handleWSGuard(c *ctx.WSContext, canActive bool) {
	if canActive {
		c.Next()
	} else {

		// TODO: handle later
		panic(exception.ForbiddenException("Access denied"))
	}
}
