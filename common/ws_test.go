package common

import (
	"testing"

	"github.com/dangduoc08/ginject/testutils"
)

func TestParseWSFnNameToEvent_Exact(t *testing.T) {
	cases := []struct {
		fn, want string
	}{
		{"ON_message", "message"},
		{"ON_chat_message", "chat.message"},
		{"ON_room_events", "room.events"},
		{"ON_a_b_c", "a.b.c"},
	}
	for _, c := range cases {
		got, ok := ParseWSFuncNameToEvent(c.fn)
		if !ok {
			t.Error(testutils.DiffMessage(ok, true, c.fn+" should be valid WS fn"))
		}
		if got != c.want {
			t.Error(testutils.DiffMessage(got, c.want, c.fn+" pattern"))
		}
	}
}

func TestParseWSFnNameToEvent_Wildcards(t *testing.T) {
	cases := []struct {
		fn, want string
	}{
		{"ON_chat_ANY", "chat.*"},
		{"ON_chat_ALL", "chat.>"},
		{"ON_ALL", ">"},
		{"ON_ANY", "*"},
	}
	for _, c := range cases {
		got, ok := ParseWSFuncNameToEvent(c.fn)
		if !ok {
			t.Error(testutils.DiffMessage(ok, true, c.fn+" should be valid WS fn"))
		}
		if got != c.want {
			t.Error(testutils.DiffMessage(got, c.want, c.fn+" pattern"))
		}
	}
}

func TestParseWSFnNameToEvent_Invalid(t *testing.T) {
	cases := []string{
		"READ_users",
		"CREATE_orders",
		"",
		"ON",
		"ON_",
		"lowercase_message",
	}
	for _, fn := range cases {
		_, ok := ParseWSFuncNameToEvent(fn)
		if ok {
			t.Error(testutils.DiffMessage(ok, false, fn+" should not be valid WS fn"))
		}
	}
}

func TestParseWSFnNameToEvent_CaseNormalization(t *testing.T) {
	got, ok := ParseWSFuncNameToEvent("ON_Chat_Message")
	if !ok {
		t.Error(testutils.DiffMessage(ok, true, "ON_Chat_Message should parse"))
	}
	if got != "chat.message" {
		t.Error(testutils.DiffMessage(got, "chat.message", "tokens lowercased"))
	}
}

func TestAddToEventMap_InitMaps(t *testing.T) {
	ws := &WS{}
	ws.addToEventMap("ON_message", "message", nil)
	if ws.EventMap == nil {
		t.Error(testutils.DiffMessage(ws.EventMap, "non-nil", "EventMap initialized"))
	}
	if ws.patternToFuncNameMap == nil {
		t.Error(testutils.DiffMessage(ws.patternToFuncNameMap, "non-nil", "patternToFuncNameMap initialized"))
	}
}

func TestAddToEventMap_StoresEntries(t *testing.T) {
	ws := &WS{}
	ws.addToEventMap("ON_message", "message", "handler")
	if ws.EventMap["message"] != "handler" {
		t.Error(testutils.DiffMessage(ws.EventMap["message"], "handler", "EventMap entry"))
	}
	if ws.patternToFuncNameMap["message"] != "ON_message" {
		t.Error(testutils.DiffMessage(ws.patternToFuncNameMap["message"], "ON_message", "patternToFuncNameMap entry"))
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

func TestAddHandlerToEventMap_WildcardStores(t *testing.T) {
	orig := InsertedEvents
	InsertedEvents = make(map[string]string)
	defer func() { InsertedEvents = orig }()

	ws := &WS{}
	ws.AddHandlerToEventMap("ON_chat_ANY", nil)
	if _, ok := ws.EventMap["chat.*"]; !ok {
		t.Error(testutils.DiffMessage("missing", "chat.*", "wildcard event stored"))
	}
	if InsertedEvents["chat.*"] != "ON_chat_ANY" {
		t.Error(testutils.DiffMessage(InsertedEvents["chat.*"], "ON_chat_ANY", "InsertedEvents wildcard"))
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
