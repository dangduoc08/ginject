package cache

import (
	"container/list"
	"testing"
)

func TestMinHeapInsert(t *testing.T) {
	l := list.New()

	l.PushFront(10)
	l.PushFront(11)

	// fmt.Println(l., l.Len())

	// size := 100
	// m := NewMinHeap(uint(size))

	// for i := 0; i < size; i++ {
	// 	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	// 	v := r.Intn(100)
	// 	m.Insert(v)
	// }

	// min := m.items[0]
	// for i := 0; i < size; i++ {
	// 	rootVal := m.items[i]
	// 	if rootVal < min {
	// 		t.Errorf("Element index 0 = %v but greater than %v at index %v", min, rootVal, i)
	// 	}
	// }
}
