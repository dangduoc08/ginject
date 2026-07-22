package common

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/str"
	"github.com/dangduoc08/ginject/routing"
)

type HTTPCanActivate = func(*ctx.HTTPContext) bool

type HTTPGuardItem struct {
	Method  string
	Route   string
	Version string
	Pattern string
	Common  CommonItem
}

func AsHTTPGuard(guarder any) (HTTPCanActivate, bool) {
	method := reflect.ValueOf(guarder).MethodByName(GuardMethodName)
	if !method.IsValid() {
		return nil, false
	}
	fn, ok := method.Interface().(HTTPCanActivate)
	return fn, ok
}

func (g *Guard) InjectProvidersIntoHTTPGuards(r *HTTP, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []GuardItem {
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

		canActivate, ok := AsHTTPGuard(guardHandler.guarder)
		if !ok {
			if _, ok = AsWSGuard(guardHandler.guarder); ok {
				continue
			}

			panic(errors.New(color.FmtRed(
				"invalid guard: %v.%s must be func(*ctx.HTTPContext) bool to be bound as a HTTP guard",
				guarderType,
				GuardMethodName,
			)))
		}

		targetedPatterns := map[string]bool{}
		for _, handler := range guardHandler.handlers {
			fnName := GetFuncName(handler)
			if pattern, ok := r.FuncNameToPatternMap[fnName]; ok {
				targetedPatterns[pattern] = true
			}
		}
		applyAll := len(targetedPatterns) == 0

		for pattern := range r.PatternToFuncNameMap {
			if applyAll || targetedPatterns[pattern] {
				method, route, version := routing.PatternToMethodRouteVersion(pattern)
				httpMethod := routing.OperationsMapHTTPMethods[method]

				guardItemArr = append(guardItemArr, GuardItem{
					HTTP: HTTPGuardItem{
						Method:  httpMethod,
						Route:   str.Enclose(route, '/'),
						Version: version,
						Pattern: pattern,
						Common: CommonItem{
							Handler:         canActivate,
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

func BuildHTTPGuardMiddleware(canActivateFn HTTPCanActivate) ctx.HTTPHandler {
	return func(c *ctx.HTTPContext) { handleHTTPGuard(c, canActivateFn(c)) }
}

func handleHTTPGuard(c *ctx.HTTPContext, canActive bool) {
	if canActive {
		c.Next()
	} else {
		panic(exception.ForbiddenException("Access denied"))
	}
}
