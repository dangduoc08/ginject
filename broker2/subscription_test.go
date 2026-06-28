package broker2

import (
	"slices"
	"sync"
	"testing"

	"github.com/dangduoc08/ginject/internal/test"
)

// ────────────────────────────────────────────────────────────────────────────
// NewSubscription
// ────────────────────────────────────────────────────────────────────────────

func TestNewSubscription_StartsEmpty(t *testing.T) {
	s := NewSubscription()

	if h := s.find("foo"); h != nil {
		t.Error(test.DiffMessage(h, nil, "find on empty subscription should return nil"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// insert + find: exact match
// ────────────────────────────────────────────────────────────────────────────

func TestInsertAndFind_ExactMatch(t *testing.T) {
	s := NewSubscription()

	var got *Message
	s.insert("foo.bar", func(m *Message) { got = m })

	handlers := s.find("foo.bar")
	if len(handlers) != 1 {
		t.Fatal(test.DiffMessage(len(handlers), 1, "handler count for newly inserted topic"))
	}

	handlers[0](&Message{Payload: "x"})
	if got == nil || got.Payload != "x" {
		t.Error(test.DiffMessage(got, "&Message{Payload: \"x\"}", "stored handler should be callable"))
	}
}

func TestInsertAndFind_NoMatch_ReturnsNil(t *testing.T) {
	s := NewSubscription()
	s.insert("foo.bar", func(*Message) {})

	if handlers := s.find("unrelated.topic"); handlers != nil {
		t.Error(test.DiffMessage(handlers, nil, "find on a topic with no subscribers should return nil"))
	}
}

func TestInsert_SameTopicTwice_AppendsHandlers(t *testing.T) {
	s := NewSubscription()
	s.insert("foo.bar", func(*Message) {})
	s.insert("foo.bar", func(*Message) {})

	handlers := s.find("foo.bar")
	if len(handlers) != 2 {
		t.Error(test.DiffMessage(len(handlers), 2, "handlers should accumulate for the same topic"))
	}
}

func TestInsert_ReturnsNonZeroID(t *testing.T) {
	s := NewSubscription()
	id := s.insert("foo", func(*Message) {})

	if id == 0 {
		t.Error(test.DiffMessage(id, "non-zero", "insert should return a non-zero subscription id"))
	}
}

func TestInsert_ReturnsUniqueIDs(t *testing.T) {
	s := NewSubscription()
	id1 := s.insert("foo", func(*Message) {})
	id2 := s.insert("foo", func(*Message) {})

	if id1 == id2 {
		t.Error(test.DiffMessage(id2, id1, "each insert should return a distinct id"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// insert + find: wildcard match
// ────────────────────────────────────────────────────────────────────────────

func TestInsertAndFind_WildcardMatch(t *testing.T) {
	s := NewSubscription()
	s.insert("foo.*", func(*Message) {})

	handlers := s.find("foo.anything")
	if len(handlers) != 1 {
		t.Error(test.DiffMessage(len(handlers), 1, "wildcard pattern should match a concrete topic"))
	}
}

func TestFind_ExactTakesPrecedenceOverWildcard(t *testing.T) {
	s := NewSubscription()

	var calledExact, calledWildcard bool
	s.insert("foo.bar", func(*Message) { calledExact = true })
	s.insert("foo.*", func(*Message) { calledWildcard = true })

	handlers := s.find("foo.bar")
	if len(handlers) != 1 {
		t.Fatal(test.DiffMessage(len(handlers), 1, "exact topic registered alongside a wildcard"))
	}
	for _, h := range handlers {
		h(&Message{})
	}

	if !calledExact || calledWildcard {
		t.Error(test.DiffMessage(
			[]bool{calledExact, calledWildcard},
			[]bool{true, false},
			"find should resolve the exact entry, not the wildcard one",
		))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// remove
// ────────────────────────────────────────────────────────────────────────────

func TestRemove_UnknownTopic_ReturnsFalse(t *testing.T) {
	s := NewSubscription()

	if ok := s.remove("never.subscribed", 1); ok {
		t.Error(test.DiffMessage(ok, false, "remove on a topic with no subscribers should report false"))
	}
}

func TestRemove_UnknownID_ReturnsFalse(t *testing.T) {
	s := NewSubscription()
	id := s.insert("foo.bar", func(*Message) {})

	if ok := s.remove("foo.bar", id+1); ok {
		t.Error(test.DiffMessage(ok, false, "remove with an id that was never issued should report false"))
	}
	if handlers := s.find("foo.bar"); len(handlers) != 1 {
		t.Error(test.DiffMessage(len(handlers), 1, "a failed remove must not mutate the topic's handlers"))
	}
}

func TestRemove_OneOfMany_RemovesOnlyThatHandler(t *testing.T) {
	s := NewSubscription()

	var calledA, calledB bool
	idA := s.insert("foo.bar", func(*Message) { calledA = true })
	s.insert("foo.bar", func(*Message) { calledB = true })

	if ok := s.remove("foo.bar", idA); !ok {
		t.Error(test.DiffMessage(ok, true, "remove of an existing handler should report true"))
	}

	handlers := s.find("foo.bar")
	if len(handlers) != 1 {
		t.Fatal(test.DiffMessage(len(handlers), 1, "handler count after removing one of two"))
	}
	for _, h := range handlers {
		h(&Message{})
	}
	if calledA || !calledB {
		t.Error(test.DiffMessage(
			[]bool{calledA, calledB},
			[]bool{false, true},
			"remove should drop only the targeted handler",
		))
	}
}

func TestRemove_LastHandler_DeletesTopic(t *testing.T) {
	s := NewSubscription()
	id := s.insert("foo.bar", func(*Message) {})

	if ok := s.remove("foo.bar", id); !ok {
		t.Error(test.DiffMessage(ok, true, "remove of the only handler should report true"))
	}
	if handlers := s.find("foo.bar"); handlers != nil {
		t.Error(test.DiffMessage(handlers, nil, "find must return nil once a topic's last handler is removed"))
	}
}

func TestRemove_AlreadyRemoved_ReturnsFalse(t *testing.T) {
	s := NewSubscription()
	id := s.insert("foo.bar", func(*Message) {})

	s.remove("foo.bar", id)
	if ok := s.remove("foo.bar", id); ok {
		t.Error(test.DiffMessage(ok, false, "second remove of the same id should report false"))
	}
}

func TestRemove_WildcardTopic_StopsMatching(t *testing.T) {
	s := NewSubscription()
	id := s.insert("foo.*", func(*Message) {})

	if handlers := s.find("foo.anything"); len(handlers) != 1 {
		t.Fatal(test.DiffMessage(len(handlers), 1, "sanity check: wildcard should match before removal"))
	}

	s.remove("foo.*", id)

	if handlers := s.find("foo.anything"); handlers != nil {
		t.Error(test.DiffMessage(handlers, nil, "removing the only wildcard subscription must stop the wildcard fast path from matching"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// list
// ────────────────────────────────────────────────────────────────────────────

func TestList_EmptySubscription_ReturnsEmptyMap(t *testing.T) {
	s := NewSubscription()

	if topics := s.list(); len(topics) != 0 {
		t.Error(test.DiffMessage(topics, map[string][]uint64{}, "list on an empty subscription should return an empty map"))
	}
}

func TestList_ListsAllTopicsAndIDs(t *testing.T) {
	s := NewSubscription()
	idA := s.insert("foo", func(*Message) {})
	idB := s.insert("foo", func(*Message) {})

	topics := s.list()
	if len(topics) != 1 {
		t.Fatal(test.DiffMessage(len(topics), 1, "list should report one entry per topic"))
	}
	ids := topics["foo"]
	if len(ids) != 2 || !slices.Contains(ids, idA) || !slices.Contains(ids, idB) {
		t.Error(test.DiffMessage(ids, []uint64{idA, idB}, "list[\"foo\"] should contain both ids"))
	}
}

func TestList_AfterRemove_DropsEmptyTopic(t *testing.T) {
	s := NewSubscription()
	id := s.insert("foo", func(*Message) {})
	s.remove("foo", id)

	if topics := s.list(); len(topics) != 0 {
		t.Error(test.DiffMessage(topics, map[string][]uint64{}, "list must not report a topic once its last handler is removed"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// find() must not leak a mutable reference (regression: handlers-slice race)
// ────────────────────────────────────────────────────────────────────────────

func TestFind_ReturnsIndependentCopy(t *testing.T) {
	s := NewSubscription()
	s.insert("foo", func(*Message) {})

	handlers := s.find("foo")
	s.insert("foo", func(*Message) {}) // append after find() returned

	if len(handlers) != 1 {
		t.Error(test.DiffMessage(len(handlers), 1, "slice returned by find() must not observe later inserts"))
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Concurrent safety (run with: go test -race ./broker/...)
// ────────────────────────────────────────────────────────────────────────────

func TestConcurrentInsertAndFind(t *testing.T) {
	s := NewSubscription()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.insert("race.topic", func(*Message) {})
		}()
		go func() {
			defer wg.Done()
			_ = s.find("race.topic")
		}()
	}
	wg.Wait()
}

func TestConcurrentInsertAndDispatch(t *testing.T) {
	s := NewSubscription()
	s.insert("race.topic", func(*Message) {})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 2)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			s.insert("race.topic", func(*Message) {})
		}()
		go func() {
			defer wg.Done()
			for _, h := range s.find("race.topic") {
				h(&Message{})
			}
		}()
	}
	wg.Wait()
}

func TestConcurrentInsertRemoveAndFind(t *testing.T) {
	s := NewSubscription()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			id := s.insert("race.topic", func(*Message) {})
			s.remove("race.topic", id)
		}()
		go func() {
			defer wg.Done()
			_ = s.find("race.topic")
		}()
		go func() {
			defer wg.Done()
			s.remove("race.topic", 0)
		}()
	}
	wg.Wait()
}
