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
			ws.AddHandlerToEventMap(fn, nil)
		}
		InsertedEvents = orig
	}
}

