package common

import "testing"

func BenchmarkAddHandlerToEventMap(b *testing.B) {
	fns := []string{
		"SUBSCRIBE_message", "SUBSCRIBE_status", "SUBSCRIBE_notification",
		"SUBSCRIBE_presence", "SUBSCRIBE_typing", "SUBSCRIBE_reaction",
		"SUBSCRIBE_thread", "SUBSCRIBE_channel", "SUBSCRIBE_direct",
		"SUBSCRIBE_group",
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
