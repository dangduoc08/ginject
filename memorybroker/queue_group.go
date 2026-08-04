package memorybroker

import "sync/atomic"

type queueGroup struct {
	subs      []*subscription
	indexByID map[string]int
	counter   atomic.Uint64
}

func newQueueGroup() *queueGroup {
	g := &queueGroup{indexByID: make(map[string]int)}
	g.counter.Store(^uint64(0))
	return g
}

func (g *queueGroup) add(sub *subscription) {
	g.indexByID[sub.id] = len(g.subs)
	g.subs = append(g.subs, sub)
}

func (g *queueGroup) remove(id string) {
	idx, ok := g.indexByID[id]
	if !ok {
		return
	}
	last := len(g.subs) - 1
	if idx != last {
		g.subs[idx] = g.subs[last]
		g.indexByID[g.subs[idx].id] = idx
	}
	g.subs[last] = nil
	g.subs = g.subs[:last]
	delete(g.indexByID, id)
}

func (g *queueGroup) pick() *subscription {
	if len(g.subs) == 0 {
		return nil
	}
	return g.subs[g.counter.Add(1)%uint64(len(g.subs))]
}
