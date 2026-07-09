package common

import (
	"errors"

	"github.com/dangduoc08/ginject/internal/color"
)

var WSOperations = map[string]string{
	"SUBSCRIBE": "SUBSCRIBE",
}

var InsertedEvents = make(map[string]string)

type WSLayer struct {
	Handler   any
	EventName string
}

type WS struct {
	funcNameByEvent map[string]string
	EventMap             map[string]any
}

func (ws *WS) addToEventMap(fnName, event string, injectableHandler any) {
	if ws.EventMap == nil {
		ws.EventMap = make(map[string]any)
	}
	if ws.funcNameByEvent == nil {
		ws.funcNameByEvent = map[string]string{}
	}
	ws.funcNameByEvent[event] = fnName
	ws.EventMap[event] = injectableHandler
}

func (ws *WS) AddHandlerToEventMap(fnName string, handler any) {
	event, ok := ParseWSFuncNameToEvent(fnName)
	if !ok {
		return
	}

	if InsertedEvents[event] == "" {
		InsertedEvents[event] = fnName
	} else {
		panic(errors.New(
			color.FmtRed(
				"%v method is conflicted with %v method",
				fnName,
				InsertedEvents[event],
			),
		))
	}

	ws.addToEventMap(fnName, event, handler)
}
