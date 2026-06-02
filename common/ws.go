package common

import (
	"errors"

	"github.com/dangduoc08/ginject/utils"
)

var WSOperations = map[string]string{
	"ON": "ON",
}

var InsertedEvents = make(map[string]string)

type WS struct {
	patternToFnNameMap map[string]string
	EventMap           map[string]any
}

func (ws *WS) addToEventMap(fnName, event string, injectableHandler any) {
	if ws.EventMap == nil {
		ws.EventMap = make(map[string]any)
	}
	if ws.patternToFnNameMap == nil {
		ws.patternToFnNameMap = map[string]string{}
	}
	ws.patternToFnNameMap[event] = fnName
	ws.EventMap[event] = injectableHandler
}

func (ws *WS) AddHandlerToEventMap(fnName string, handler any) {
	event, ok := ParseWSFnNameToEvent(fnName)
	if !ok {
		return
	}

	if InsertedEvents[event] == "" {
		InsertedEvents[event] = fnName
	} else {
		panic(errors.New(
			utils.FmtRed(
				"%v method is conflicted with %v method",
				fnName,
				InsertedEvents[event],
			),
		))
	}

	ws.addToEventMap(fnName, event, handler)
}
