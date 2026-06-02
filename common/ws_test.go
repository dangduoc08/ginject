package common

import (
	"testing"

	"github.com/dangduoc08/ginject/testutils"
)

func TestAddToEventMap_InitMaps(t *testing.T) {
	ws := &WS{}
	ws.addToEventMap("ON_message", "chat_message", nil)
	if ws.EventMap == nil {
		t.Error(testutils.DiffMessage(ws.EventMap, "non-nil", "EventMap initialized"))
	}
	if ws.patternToFnNameMap == nil {
		t.Error(testutils.DiffMessage(ws.patternToFnNameMap, "non-nil", "patternToFnNameMap initialized"))
	}
}

func TestAddToEventMap_StoresEntries(t *testing.T) {
	ws := &WS{}
	ws.addToEventMap("ON_message", "chat_message", "handler")
	if ws.EventMap["chat_message"] != "handler" {
		t.Error(testutils.DiffMessage(ws.EventMap["chat_message"], "handler", "EventMap entry"))
	}
	if ws.patternToFnNameMap["chat_message"] != "ON_message" {
		t.Error(testutils.DiffMessage(ws.patternToFnNameMap["chat_message"], "ON_message", "patternToFnNameMap entry"))
	}
}

func TestAddHandlerToEventMap_Stores(t *testing.T) {
	orig := InsertedEvents
	InsertedEvents = make(map[string]string)
	defer func() { InsertedEvents = orig }()

	ws := &WS{}
	ws.AddHandlerToEventMap("ON_message", nil)
	if ws.EventMap == nil {
		t.Error(testutils.DiffMessage(ws.EventMap, "non-nil", "EventMap created"))
		return
	}
	eventName := "message"
	if _, ok := ws.EventMap[eventName]; !ok {
		t.Error(testutils.DiffMessage("missing", eventName, "event stored in EventMap"))
	}
	if InsertedEvents[eventName] != "ON_message" {
		t.Error(testutils.DiffMessage(InsertedEvents[eventName], "ON_message", "InsertedEvents entry"))
	}
}

func TestAddHandlerToEventMap_IgnoresNonWS(t *testing.T) {
	orig := InsertedEvents
	InsertedEvents = make(map[string]string)
	defer func() { InsertedEvents = orig }()

	ws := &WS{}
	ws.AddHandlerToEventMap("READ_users", nil)
	if len(ws.EventMap) != 0 {
		t.Error(testutils.DiffMessage(len(ws.EventMap), 0, "non-WS fn should not be stored"))
	}
}

func TestAddHandlerToEventMap_Conflict(t *testing.T) {
	orig := InsertedEvents
	InsertedEvents = make(map[string]string)
	defer func() { InsertedEvents = orig }()

	ws := &WS{}
	ws.AddHandlerToEventMap("ON_message", nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "duplicate event should panic"))
		}
	}()
	ws.AddHandlerToEventMap("ON_message", nil)
}

