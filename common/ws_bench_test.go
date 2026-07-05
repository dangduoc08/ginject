package common

import "testing"

func BenchmarkAddHandlerToEventMap(b *testing.B) {
	fns := []string{
		"SUBSCRIBE_chat_message", "SUBSCRIBE_chat_status", "SUBSCRIBE_chat_ANY",
		"SUBSCRIBE_room_join", "SUBSCRIBE_room_leave", "SUBSCRIBE_room_ALL",
		"SUBSCRIBE_user_typing", "SUBSCRIBE_user_presence", "SUBSCRIBE_notification",
		"SUBSCRIBE_ALL",
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
		"SUBSCRIBE_chat_message", "SUBSCRIBE_chat_ANY", "SUBSCRIBE_chat_ALL", "SUBSCRIBE_ALL",
	}
	b.ResetTimer()
	for range b.N {
		for _, fn := range fns {
			ParseWSFuncNameToEvent(fn)
		}
	}
}
