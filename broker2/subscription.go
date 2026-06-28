package broker2

import (
	"strings"
	"sync"
	"sync/atomic"

	"github.com/dangduoc08/ginject/internal/ds"
	"github.com/dangduoc08/ginject/internal/str"
)

type Message struct {
	Topic   string
	Payload any
}

type MessageHandler func(*Message)

type subEntry struct {
	id      uint64
	handler MessageHandler
}

type SubscriptionItem struct {
	entries []subEntry
}

type Subscription struct {
	rwMu          sync.RWMutex
	trie          *ds.Trie
	hash          map[string]*SubscriptionItem
	nextID        atomic.Uint64
	wildcardCount int
}

func NewSubscription() *Subscription {
	return &Subscription{
		trie: ds.NewTrie(),
		hash: make(map[string]*SubscriptionItem),
	}
}

func (t *Subscription) insert(topic string, handler MessageHandler) uint64 {
	t.rwMu.Lock()
	defer t.rwMu.Unlock()

	id := t.nextID.Add(1)

	if item, ok := t.hash[topic]; ok {
		item.entries = append(item.entries, subEntry{id: id, handler: handler})
	} else {
		t.trie.Insert(topic, str.Enclose(topic, '.'), '.')
		if strings.ContainsRune(topic, '*') {
			t.wildcardCount++
		}
		t.hash[topic] = &SubscriptionItem{
			entries: []subEntry{{id: id, handler: handler}},
		}
	}

	return id
}

func (t *Subscription) remove(topic string, id uint64) bool {
	t.rwMu.Lock()
	defer t.rwMu.Unlock()

	item, ok := t.hash[topic]
	if !ok {
		return false
	}

	idx := -1
	for i := range item.entries {
		if item.entries[i].id == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}

	last := len(item.entries) - 1
	item.entries[idx] = item.entries[last]
	item.entries = item.entries[:last]

	if len(item.entries) == 0 {
		delete(t.hash, topic)
		t.trie.Remove(str.Enclose(topic, '.'), '.')
		if strings.ContainsRune(topic, '*') {
			t.wildcardCount--
		}
	}

	return true
}

func (t *Subscription) list() map[string][]uint64 {
	t.rwMu.RLock()
	defer t.rwMu.RUnlock()

	if len(t.hash) == 0 {
		return nil
	}

	result := make(map[string][]uint64, len(t.hash))
	for topic, item := range t.hash {
		ids := make([]uint64, len(item.entries))
		for i := range item.entries {
			ids[i] = item.entries[i].id
		}
		result[topic] = ids
	}
	return result
}

func (t *Subscription) find(topic string) []MessageHandler {
	t.rwMu.RLock()
	defer t.rwMu.RUnlock()

	item, ok := t.hash[topic]
	if !ok {
		if t.wildcardCount == 0 {
			return nil
		}
		_, wildcardRaw, _ := t.trie.Find(str.Enclose(topic, '.'), '.', false)
		item, ok = t.hash[wildcardRaw]
		if !ok {
			return nil
		}
	}

	handlers := make([]MessageHandler, len(item.entries))
	for i := range item.entries {
		handlers[i] = item.entries[i].handler
	}
	return handlers
}
