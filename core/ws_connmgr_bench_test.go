package core

import (
	"testing"

	"github.com/dangduoc08/ginject/log"
)

func BenchmarkWSConnmgr_Get(b *testing.B) {
	connmgr := NewWSConnmgr(log.NewLog(nil))
	connmgr.conns["conn-1"] = &WSConnection{ID: "conn-1"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connmgr.Get("conn-1")
	}
}

func BenchmarkWSConnmgr_Touch(b *testing.B) {
	connmgr := NewWSConnmgr(log.NewLog(nil))
	connmgr.conns["conn-1"] = &WSConnection{ID: "conn-1"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connmgr.touch("conn-1")
	}
}

func BenchmarkWSConnmgr_IsSubscribed(b *testing.B) {
	connmgr := NewWSConnmgr(log.NewLog(nil))
	connmgr.subscriptions["conn-1"] = []wsSubscription{{topic: "a"}, {topic: "b"}, {topic: "c"}}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connmgr.isSubscribed("conn-1", "c")
	}
}
