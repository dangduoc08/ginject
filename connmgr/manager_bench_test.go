package connmgr

import (
	"fmt"
	"testing"
)

func BenchmarkConnectionManager_Add(b *testing.B) {
	m := NewConnectionManager()
	conns := make([]*Connection, b.N)
	for i := range b.N {
		conns[i] = newTestConn("user1")
	}
	b.ResetTimer()
	for i := range b.N {
		m.Add(conns[i])
	}
}

func BenchmarkConnectionManager_Get(b *testing.B) {
	m := NewConnectionManager()
	c := newTestConn("user1")
	m.Add(c)
	b.ResetTimer()
	for range b.N {
		m.Get(c.ID)
	}
}

func BenchmarkConnectionManager_Remove(b *testing.B) {
	m := NewConnectionManager()
	conns := make([]*Connection, b.N)
	for i := range b.N {
		conns[i] = newTestConn("user1")
		m.Add(conns[i])
	}
	b.ResetTimer()
	for i := range b.N {
		m.Remove(conns[i].ID)
	}
}

func BenchmarkConnectionManager_Count(b *testing.B) {
	m := NewConnectionManager()
	for range 1000 {
		m.Add(newTestConn("user1"))
	}
	b.ResetTimer()
	for range b.N {
		m.Count()
	}
}

func BenchmarkConnectionManager_Connections_1k(b *testing.B) {
	m := NewConnectionManager()
	for range 1000 {
		m.Add(newTestConn("user1"))
	}
	b.ResetTimer()
	for range b.N {
		m.Connections()
	}
}

func BenchmarkConnectionManager_GetByUser_100conns(b *testing.B) {
	m := NewConnectionManager()
	for range 100 {
		m.Add(newTestConn("alice"))
	}
	b.ResetTimer()
	for range b.N {
		m.GetByUser("alice")
	}
}

func BenchmarkConnectionManager_ParallelGet(b *testing.B) {
	m := NewConnectionManager()
	c := newTestConn("user1")
	m.Add(c)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Get(c.ID)
		}
	})
}

func BenchmarkConnectionManager_ParallelAddRemove(b *testing.B) {
	m := NewConnectionManager()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c := newTestConn("user1")
			m.Add(c)
			m.Remove(c.ID)
		}
	})
}

func BenchmarkConnectionManager_MultiUser(b *testing.B) {
	m := NewConnectionManager()
	users := make([]string, 100)
	for i := range 100 {
		users[i] = fmt.Sprintf("user%d", i)
		for range 10 {
			m.Add(newTestConn(users[i]))
		}
	}
	b.ResetTimer()
	for i := range b.N {
		m.GetByUser(users[i%100])
	}
}
