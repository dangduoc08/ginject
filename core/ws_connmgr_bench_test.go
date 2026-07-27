package core

import (
	"testing"

	"github.com/dangduoc08/ginject/broker"
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
	noop := func(*broker.Message) {}
	for _, topic := range []string{"a", "b", "c"} {
		if err := connmgr.Subscribe("conn-1", topic, noop); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connmgr.isSubscribed("conn-1", "c")
	}
}
