package core

import (
	"testing"

	"github.com/dangduoc08/ginject/ctx"
)

func BenchmarkMatchEventKey_Exact(b *testing.B) {
	ws := buildTestWS([]string{"chat.message", "chat.status", "room.join", "room.leave", "notification"})
	b.ResetTimer()
	for range b.N {
		ws.matchEventKey("chat.message")
	}
}

func BenchmarkMatchEventKey_Wildcard(b *testing.B) {
	ws := buildTestWS([]string{"chat.*", "room.*", "notification"})
	b.ResetTimer()
	for range b.N {
		ws.matchEventKey("chat.message")
	}
}

func BenchmarkMatchEventKey_CatchAll(b *testing.B) {
	ws := buildTestWS([]string{">"})
	b.ResetTimer()
	for range b.N {
		ws.matchEventKey("chat.message.deep")
	}
}

func BenchmarkBuildCompiledPatterns(b *testing.B) {
	patterns := []string{
		"chat.message", "chat.status", "chat.*",
		"room.join", "room.leave", "room.>",
		"notification", ">",
	}
	b.ResetTimer()
	for range b.N {
		ws := &WS{eventMap: make(map[string][]ctx.Handler)}
		for _, p := range patterns {
			ws.eventMap[p] = nil
		}
		ws.buildCompiledPatterns()
	}
}
