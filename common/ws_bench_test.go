package common

import "testing"

func BenchmarkAddHandlerToEventMap(b *testing.B) {
	fns := []string{
		"ON_chat_message", "ON_chat_status", "ON_chat_ANY",
		"ON_room_join", "ON_room_leave", "ON_room_ALL",
		"ON_user_typing", "ON_user_presence", "ON_notification",
		"ON_ALL",
	}
	b.ResetTimer()
	for range b.N {
		orig := InsertedEvents
		InsertedEvents = make(map[string]string)
		ws := &WS{}
		for _, fn := range fns {
			ws.AddHandlerToEventMap(fn, nil)
		}
		InsertedEvents = orig
	}
}

func BenchmarkWSFnToSubject(b *testing.B) {
	fns := []string{
		"ON_chat_message", "ON_chat_ANY", "ON_chat_ALL", "ON_ALL",
	}
	b.ResetTimer()
	for range b.N {
		for _, fn := range fns {
			ParseWSFuncNameToEvent(fn)
		}
	}
}
