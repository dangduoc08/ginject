package connmgr

import (
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

func TestNewConnectionManager(t *testing.T) {
	m := NewConnectionManager()
	if m == nil {
		t.Fatal("NewConnectionManager returned nil")
	}
	if m.Count() != 0 {
		t.Error(test.DiffMessage(m.Count(), 0, "initial count"))
	}
	if conns := m.Connections(); len(conns) != 0 {
		t.Error(test.DiffMessage(len(conns), 0, "initial Connections()"))
	}
}

func TestConnectionManager_Add_Get(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("user1")
	m.Add(c)

	got, ok := m.Get(c.ID)
	if !ok {
		t.Error(test.DiffMessage(ok, true, "Get must find added connection"))
	}
	if got != c {
		t.Error(test.DiffMessage(got, c, "Get must return same pointer"))
	}
}

func TestConnectionManager_Get_NotFound(t *testing.T) {
	m := NewConnectionManager()
	_, ok := m.Get("nonexistent")
	if ok {
		t.Error(test.DiffMessage(ok, false, "Get on unknown ID must return false"))
	}
}

func TestConnectionManager_Remove(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("user1")
	m.Add(c)
	m.Remove(c.ID)

	_, ok := m.Get(c.ID)
	if ok {
		t.Error(test.DiffMessage(ok, false, "Get must not find removed connection"))
	}
	if m.Count() != 0 {
		t.Error(test.DiffMessage(m.Count(), 0, "count after remove"))
	}
}

func TestConnectionManager_Remove_NonExistent(t *testing.T) {
	m := NewConnectionManager()
	m.Remove("never-added")
	if m.Count() != 0 {
		t.Error(test.DiffMessage(m.Count(), 0, "count must stay 0"))
	}
}

func TestConnectionManager_Remove_Idempotent(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("")
	m.Add(c)
	m.Remove(c.ID)
	m.Remove(c.ID)
	if m.Count() != 0 {
		t.Error(test.DiffMessage(m.Count(), 0, "double remove must not underflow count"))
	}
}

func TestConnectionManager_Exists(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("")
	if m.Exists(c.ID) {
		t.Error(test.DiffMessage(true, false, "Exists before Add"))
	}
	m.Add(c)
	if !m.Exists(c.ID) {
		t.Error(test.DiffMessage(false, true, "Exists after Add"))
	}
	m.Remove(c.ID)
	if m.Exists(c.ID) {
		t.Error(test.DiffMessage(true, false, "Exists after Remove"))
	}
}

func TestConnectionManager_Count(t *testing.T) {
	m := NewConnectionManager()
	conns := []*Connection{newTestConn(""), newTestConn(""), newTestConn("")}
	for _, c := range conns {
		m.Add(c)
	}
	if m.Count() != 3 {
		t.Error(test.DiffMessage(m.Count(), 3, "Count after 3 adds"))
	}
	m.Remove(conns[0].ID)
	if m.Count() != 2 {
		t.Error(test.DiffMessage(m.Count(), 2, "Count after 1 remove"))
	}
}

func TestConnectionManager_Connections_Snapshot(t *testing.T) {
	m := NewConnectionManager()
	a, b := newTestConn(""), newTestConn("")
	m.Add(a)
	m.Add(b)

	snap := m.Connections()
	if len(snap) != 2 {
		t.Error(test.DiffMessage(len(snap), 2, "Connections() length"))
	}

	m.Remove(a.ID)
	if len(m.Connections()) != 1 {
		t.Error(test.DiffMessage(len(m.Connections()), 1, "Connections() after remove"))
	}
	if len(snap) != 2 {
		t.Error(test.DiffMessage(len(snap), 2, "old snapshot must not change"))
	}
}

func TestConnectionManager_Anonymous_NotInUserIndex(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("")
	m.Add(c)

	byUser := m.GetByUser("")
	if len(byUser) != 0 {
		t.Error(test.DiffMessage(len(byUser), 0, "anonymous connections must not appear in user index"))
	}
	if m.Count() != 1 {
		t.Error(test.DiffMessage(m.Count(), 1, "anonymous connection still counted"))
	}
}

func TestConnectionManager_GetByUser(t *testing.T) {
	m := NewConnectionManager()
	a, b, c := newTestConn("alice"), newTestConn("alice"), newTestConn("bob")
	m.Add(a)
	m.Add(b)
	m.Add(c)

	aliceConns := m.GetByUser("alice")
	if len(aliceConns) != 2 {
		t.Error(test.DiffMessage(len(aliceConns), 2, "alice should have 2 connections"))
	}
	bobConns := m.GetByUser("bob")
	if len(bobConns) != 1 {
		t.Error(test.DiffMessage(len(bobConns), 1, "bob should have 1 connection"))
	}
}

func TestConnectionManager_GetByUser_NotFound(t *testing.T) {
	m := NewConnectionManager()
	conns := m.GetByUser("nobody")
	if conns == nil || len(conns) != 0 {
		t.Error(test.DiffMessage(len(conns), 0, "GetByUser unknown user must return empty slice"))
	}
}

func TestConnectionManager_GetByUser_CleanedOnLastRemove(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("alice")
	m.Add(c)
	m.Remove(c.ID)

	m.mu.RLock()
	_, exists := m.byUser["alice"]
	m.mu.RUnlock()
	if exists {
		t.Error(test.DiffMessage(true, false, "user entry must be removed when last connection leaves"))
	}
}

func TestConnectionManager_MultipleUsers(t *testing.T) {
	m := NewConnectionManager()
	for i := range 5 {
		userID := "user"
		switch i {
		case 0, 1:
			userID = "alice"
		case 2, 3, 4:
			userID = "bob"
		}
		m.Add(newTestConn(userID))
	}

	if len(m.GetByUser("alice")) != 2 {
		t.Error(test.DiffMessage(len(m.GetByUser("alice")), 2, "alice connection count"))
	}
	if len(m.GetByUser("bob")) != 3 {
		t.Error(test.DiffMessage(len(m.GetByUser("bob")), 3, "bob connection count"))
	}
	if m.Count() != 5 {
		t.Error(test.DiffMessage(m.Count(), 5, "total count"))
	}
}

func TestConnectionManager_Concurrent_AddRemove(t *testing.T) {
	m := NewConnectionManager()
	const n = 1000

	conns := make([]*Connection, n)
	for i := range n {
		conns[i] = newTestConn("user")
	}

	var wg sync.WaitGroup
	for _, c := range conns {
		c := c
		wg.Add(2)
		go func() {
			defer wg.Done()
			m.Add(c)
		}()
		go func() {
			defer wg.Done()
			m.Remove(c.ID)
		}()
	}
	wg.Wait()
}

func TestConnectionManager_Concurrent_GetExists(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("alice")
	m.Add(c)

	var wg sync.WaitGroup
	for range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Get(c.ID)
			m.Exists(c.ID)
			m.Count()
			m.GetByUser("alice")
			m.Connections()
		}()
	}
	wg.Wait()
}

func TestConnectionManager_Connections_EmptyManager(t *testing.T) {
	m := NewConnectionManager()
	conns := m.Connections()
	if len(conns) != 0 {
		t.Error(test.DiffMessage(len(conns), 0, "Connections() on empty manager"))
	}
}

func TestConnectionManager_Add_Duplicate(t *testing.T) {
	m := NewConnectionManager()
	c := newTestConn("alice")
	m.Add(c)
	m.Add(c)
	m.Add(c)

	if m.Count() != 1 {
		t.Error(test.DiffMessage(m.Count(), 1, "duplicate Add must not increment count"))
	}
	if len(m.GetByUser("alice")) != 1 {
		t.Error(test.DiffMessage(len(m.GetByUser("alice")), 1, "duplicate Add must not inflate user index"))
	}
	if len(m.Connections()) != 1 {
		t.Error(test.DiffMessage(len(m.Connections()), 1, "duplicate Add must not inflate Connections()"))
	}
}

func TestConnectionManager_GetByUser_AfterRemoveOne(t *testing.T) {
	m := NewConnectionManager()
	a, b := newTestConn("alice"), newTestConn("alice")
	m.Add(a)
	m.Add(b)
	m.Remove(a.ID)

	conns := m.GetByUser("alice")
	if len(conns) != 1 {
		t.Error(test.DiffMessage(len(conns), 1, "alice should have 1 connection after remove"))
	}
	if conns[0].ID != b.ID {
		t.Error(test.DiffMessage(conns[0].ID, b.ID, "remaining connection ID"))
	}
}
