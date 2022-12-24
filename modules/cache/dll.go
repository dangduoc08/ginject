package cache

import (
	"sync/atomic"
)

type node[T any] struct {
	next *node[T]
	prev *node[T]
	data T
}

type DLL[T any] struct {
	head *node[T]
	tail *node[T]
	size int64
}

func NewDLL[T any]() *DLL[T] {
	return &DLL[T]{
		head: nil,
		tail: nil,
		size: 0,
	}
}

// Add at tail of list
func (dll *DLL[T]) push(d T) *node[T] {
	n := &node[T]{
		next: nil,
		prev: dll.tail,
		data: d,
	}

	if dll.size >= 1 {
		if dll.tail != nil {
			dll.tail.next = n
		} else {
			dll.tail = n
			dll.tail.prev = dll.head
			dll.head.next = dll.tail
		}

		if dll.head == nil {
			dll.head = dll.tail
		}
	}

	dll.tail = n
	atomic.AddInt64(&dll.size, 1)

	return n
}

// Add at head of list
func (dll *DLL[T]) unshift(d T) *node[T] {
	n := &node[T]{
		next: dll.head,
		prev: nil,
		data: d,
	}

	if dll.size >= 1 {
		if dll.head != nil {
			dll.head.prev = n
		} else {
			dll.head = n
			dll.head.next = dll.tail
			dll.tail.prev = dll.head
		}

		if dll.tail == nil {
			dll.tail = dll.head
		}
	}

	dll.head = n
	atomic.AddInt64(&dll.size, 1)

	return n
}

// Remove at tail of list
func (dll *DLL[T]) pop() *node[T] {
	n := dll.tail
	if n == nil {
		if dll.head != nil {
			return dll.shift()
		}
		return nil
	}
	atomic.AddInt64(&dll.size, -1)

	if dll.size > 0 {
		dll.tail = dll.tail.prev
		dll.tail.next = nil
		if dll.size == 1 {
			dll.tail = nil
		}
	} else {
		dll.tail = nil
	}

	return n
}

// Add at head of list
func (dll *DLL[T]) shift() *node[T] {
	n := dll.head
	if n == nil {
		if dll.tail != nil {
			return dll.pop()
		}
		return nil
	}
	atomic.AddInt64(&dll.size, -1)

	if dll.size > 0 {
		dll.head = dll.head.next
		dll.head.prev = nil
		if dll.size == 1 {
			dll.head = nil
		}
	} else {
		dll.head = nil
	}

	return n
}

func (dll *DLL[T]) delete(n *node[T]) {
	if dll.size == 0 {
		return
	}

	if dll.size == 1 {
		if dll.head != nil {
			dll.head = nil
		} else if dll.tail != nil {
			dll.tail = nil
		}
		n = nil
		dll.size = 0
		return
	}

	if n == dll.head {
		dll.shift()
		return
	}

	if n == dll.tail {
		dll.pop()
		return
	}

	n.next.prev = n.prev
	n.prev.next = n.next
	n = nil
	atomic.AddInt64(&dll.size, -1)
}
