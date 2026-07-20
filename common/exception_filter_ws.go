package common

import (
	"errors"
	"reflect"

	"github.com/dangduoc08/ginject/ctx"
	"github.com/dangduoc08/ginject/exception"
	"github.com/dangduoc08/ginject/internal/color"
)

type WSCatch = func(*ctx.WSContext, *exception.Exception)

type WSExceptionFilterItem struct {
	EventName string
	Common    CommonItem
}

func AsWSExceptionFilter(exceptionFilterable any) (WSCatch, bool) {
	method := reflect.ValueOf(exceptionFilterable).MethodByName(ExceptionFilterMethodName)
	if !method.IsValid() {
		return nil, false
	}
	fn, ok := method.Interface().(WSCatch)
	return fn, ok
}

func (e *ExceptionFilter) InjectProvidersIntoWSExceptionFilters(ws *WS, cb func(int, reflect.Type, reflect.Value, reflect.Value)) []ExceptionFilterItem {
	exceptionFilterItemArr := make([]ExceptionFilterItem, 0, len(ws.funcNameByEvent)*len(e.ExceptionFilterHandlers))

	for _, exceptionFilterHandler := range e.ExceptionFilterHandlers {
		exceptionFilterableType := reflect.TypeOf(exceptionFilterHandler.exceptionFilterable)
		exceptionFilterableValue := reflect.ValueOf(exceptionFilterHandler.exceptionFilterable)
		newExceptionFilter := reflect.New(exceptionFilterableType)

		for i := 0; i < exceptionFilterableType.NumField(); i++ {
			cb(i, exceptionFilterableType, exceptionFilterableValue, newExceptionFilter)
		}

		newExceptionFilterable := newExceptionFilter.Interface()
		newExceptionFilterable = Construct(newExceptionFilterable, "NewExceptionFilter")
		exceptionFilterHandler.exceptionFilterable = newExceptionFilterable.(ExceptionFilterable)

		catch, ok := AsWSExceptionFilter(exceptionFilterHandler.exceptionFilterable)
		if !ok {
			if _, ok = AsRESTExceptionFilter(exceptionFilterHandler.exceptionFilterable); ok {
				continue
			}

			panic(errors.New(color.FmtRed(
				"invalid exception filter: %v.%s must be func(*ctx.WSContext, *exception.Exception) to be bound as a WS exception filter",
				exceptionFilterableType,
				ExceptionFilterMethodName,
			)))
		}

		targetedPatterns := map[string]bool{}
		for _, handler := range exceptionFilterHandler.handlers {
			fnName := GetFuncName(handler)
			if event, ok := ParseWSFuncNameToEvent(fnName); ok {
				targetedPatterns[event] = true
			}
		}
		applyAll := len(targetedPatterns) == 0

		for pattern := range ws.funcNameByEvent {
			if applyAll || targetedPatterns[pattern] {
				exceptionFilterItemArr = append(exceptionFilterItemArr, ExceptionFilterItem{
					WS: WSExceptionFilterItem{
						EventName: pattern,
						Common: CommonItem{
							Handler: catch,
							Name:    exceptionFilterableType.String(),
						},
					},
				})
			}
		}
	}

	return exceptionFilterItemArr
}

func BuildWSCatchMiddleware(catchEvent string, catchFns []WSCatch) ctx.WSHandler {
	return func(c *ctx.WSContext) {
		c.Event.On(catchEvent, func(args ...any) {
			p := args[0].(CatchEventPayload)
			catchFnIndex := p.Index

			defer func() {
				if rec := recover(); rec != nil {
					c.Event.Emit(catchEvent, CatchEventPayload{ReqCtx: p.ReqCtx, Recovered: rec, Index: catchFnIndex + 1})
				}
			}()

			catchFns[catchFnIndex](p.ReqCtx.(*ctx.WSContext), NormalizeRecovered(p.Recovered))
		})

		c.Next()
	}
}
