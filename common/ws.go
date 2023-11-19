package common

import (
	"fmt"
	"reflect"

	"github.com/dangduoc08/gooh/utils"
)

var WSOperations = map[string]string{
	"SUBSCRIBE": "SUBSCRIBE",
}

type WS struct {
	patternToFnNameMap map[string]string
	EventMap           map[string]any
	subprotocol        string
}

func (ws *WS) addToEventMap(fnName, event string, injectableHandler any) {
	if reflect.ValueOf(ws.EventMap).IsNil() {
		ws.EventMap = make(map[string]any)
	}

	if ws.patternToFnNameMap == nil {
		ws.patternToFnNameMap = map[string]string{}
	}

	ws.patternToFnNameMap[event] = fnName
	ws.EventMap[event] = injectableHandler
}

func (ws *WS) AddHandlerToEventMap(subprotocol string, fnName string, insertedEvents map[string]string, handler any) {
	opr, eventName := ParseFnNameToURL(fnName, WSOperations)

	if opr != "" && WSOperations[opr] != "" {
		eventName = ToWSEventName(subprotocol, eventName)

		if insertedEvents[eventName] == "" {
			insertedEvents[eventName] = fnName
		} else {
			panic(fmt.Errorf(
				utils.FmtRed(
					"%v method is conflicted with %v method",
					fnName,
					insertedEvents[eventName],
				),
			))
		}

		ws.addToEventMap(fnName, eventName, handler)
	}
}

func (ws *WS) Subprotocol(p string) *WS {
	ws.subprotocol = p
	return ws
}

func (ws *WS) GetSubprotocol() string {
	if ws.subprotocol == "" {
		return "*"
	}
	return ws.subprotocol
}
