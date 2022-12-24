package cache

import (
	"sync/atomic"
)

type MinHeap struct {
	items []int
	len   int64
	size  uint
}

func NewMinHeap(size uint) *MinHeap {
	return &MinHeap{
		items: make([]int, size),
		len:   0,
		size:  size,
	}
}

// formulations:
// root: get root from child => (child - 1) / 2
// left: odd num => get left from root: (root * 2) + 1
// right: even num => => get right from root: (root * 2) + 2
func (m *MinHeap) getRoot(i int) int {
	if i == 0 {
		return 0
	}

	return (i - 1) / 2
}

func (m *MinHeap) getLeft(i int) int {
	return (i * 2) + 1
}

func (m *MinHeap) getRight(i int) int {
	return (i * 2) + 2
}

func (m *MinHeap) Insert(v int) *MinHeap {
	if uint(m.len) == m.size {
		return m
	}

	curIndex := int(m.len)
	rootIndex := m.getRoot(curIndex)
	rootVal := m.items[m.getRoot(curIndex)]

	if curIndex == 0 || v >= rootVal {
		m.items[curIndex] = v
		atomic.AddInt64(&m.len, 1)
		return m
	}

	for v < rootVal {
		m.items[curIndex], m.items[rootIndex] = rootVal, v

		v = m.items[rootIndex]
		curIndex = rootIndex
		rootIndex = m.getRoot(rootIndex)
		rootVal = m.items[rootIndex]
	}

	atomic.AddInt64(&m.len, 1)
	return m
}

func (m *MinHeap) Remove(i int) *MinHeap {

	return m
}
