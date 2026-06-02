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
	ws.AddHandlerToEventMap("chat", "ON_message", nil)
	if ws.EventMap == nil {
		t.Error(testutils.DiffMessage(ws.EventMap, "non-nil", "EventMap created"))
		return
	}
	eventName := "chat_message"
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
	ws.AddHandlerToEventMap("chat", "READ_users", nil)
	if len(ws.EventMap) != 0 {
		t.Error(testutils.DiffMessage(len(ws.EventMap), 0, "non-WS fn should not be stored"))
	}
}

func TestAddHandlerToEventMap_Conflict(t *testing.T) {
	orig := InsertedEvents
	InsertedEvents = make(map[string]string)
	defer func() { InsertedEvents = orig }()

	ws := &WS{}
	ws.AddHandlerToEventMap("chat", "ON_message", nil)
	defer func() {
		if r := recover(); r == nil {
			t.Error(testutils.DiffMessage(nil, "panic", "duplicate event should panic"))
		}
	}()
	ws.AddHandlerToEventMap("chat", "ON_message", nil)
}

func TestSubprotocol_SetAndGet(t *testing.T) {
	ws := &WS{}
	ret := ws.Subprotocol("myproto")
	if ret != ws {
		t.Error(testutils.DiffMessage(ret, ws, "Subprotocol should return self"))
	}
	if ws.GetSubprotocol() != "myproto" {
		t.Error(testutils.DiffMessage(ws.GetSubprotocol(), "myproto", "GetSubprotocol after set"))
	}
}

func TestGetSubprotocol_Default(t *testing.T) {
	ws := &WS{}
	got := ws.GetSubprotocol()
	if got != "*" {
		t.Error(testutils.DiffMessage(got, "*", "default subprotocol is *"))
	}
}
