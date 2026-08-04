package memorybroker

import (
	"github.com/dangduoc08/ginject/internal/color"
	"github.com/dangduoc08/ginject/internal/crypto"
	ptrn "github.com/dangduoc08/ginject/pattern"
)

func newID() string {
	id, err := crypto.UUID()
	if err != nil {
		panic(color.FmtRed("broker: failed to generate UUID: %v", err))
	}
	return id
}

func forEachPrefixOf(topic string, fn func(prefix string)) {
	for i := len(topic) - 1; i >= 0; i-- {
		if topic[i] == '.' {
			fn(topic[:i])
		}
	}
}

func (b *MemoryBroker) removeFromBucket(sub *subscription) {
	switch sub.pattern.Kind() {
	case ptrn.KindGlobal:
		delete(b.globalByID, sub.id)
	case ptrn.KindSuffixWildcard:
		pfx := sub.pattern.SimplePrefix()
		delete(b.prefixByPrefix[pfx], sub.id)
		if len(b.prefixByPrefix[pfx]) == 0 {
			delete(b.prefixByPrefix, pfx)
		}
	case ptrn.KindExact:
		delete(b.exactByTopic[sub.topic], sub.id)
		if len(b.exactByTopic[sub.topic]) == 0 {
			delete(b.exactByTopic, sub.topic)
		}
	case ptrn.KindComplex:
		if cg := b.complexByTopic[sub.topic]; cg != nil {
			delete(cg.subsByID, sub.id)
			if len(cg.subsByID) == 0 {
				delete(b.complexByTopic, sub.topic)
			}
		}
	}
}
