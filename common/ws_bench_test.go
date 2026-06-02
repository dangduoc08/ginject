package common

import "testing"

func BenchmarkAddHandlerToEventMap(b *testing.B) {
	fns := []string{
		"ON_message", "ON_status", "ON_notification",
		"ON_presence", "ON_typing", "ON_reaction",
		"ON_thread", "ON_channel", "ON_direct",
		"ON_group",
	}
	b.ResetTimer()
	for range b.N {
		orig := InsertedEvents
		InsertedEvents = make(map[string]string)
		ws := &WS{}
		for _, fn := range fns {
			ws.AddHandlerToEventMap("chat", fn, nil)
		}
		InsertedEvents = orig
	}
}

func BenchmarkGetSubprotocol_Set(b *testing.B) {
	ws := &WS{}
	ws.Subprotocol("myproto")
	b.ResetTimer()
	for range b.N {
		_ = ws.GetSubprotocol()
	}
}

func BenchmarkGetSubprotocol_Default(b *testing.B) {
	ws := &WS{}
	b.ResetTimer()
	for range b.N {
		_ = ws.GetSubprotocol()
	}
}
