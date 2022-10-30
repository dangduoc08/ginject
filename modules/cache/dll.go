package cache

type node[T any] struct {
	next *node[T]
	prev *node[T]
	data T
}

type dll[T any] struct {
	head *node[T]
	tail *node[T]
	size int
}

func newDLL[T any]() *dll[T] {
	return &dll[T]{
		head: nil,
		tail: nil,
		size: 0,
	}
}

func (dll *dll[T]) push(d T) *node[T] {
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
	dll.size += 1

	return n
}

func (dll *dll[T]) unshift(d T) *node[T] {
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
	dll.size += 1

	return n
}

func (dll *dll[T]) pop() *node[T] {
	n := dll.tail
	if n == nil {
		if dll.head != nil {
			return dll.shift()
		}
		return nil
	}
	dll.size -= 1

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

func (dll *dll[T]) shift() *node[T] {
	n := dll.head
	if n == nil {
		if dll.tail != nil {
			return dll.pop()
		}
		return nil
	}
	dll.size -= 1

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

func (dll *dll[T]) delete(n *node[T]) *node[T] {
	if dll.size == 0 {
		return nil
	}

	if dll.size == 1 {
		if dll.head != nil {
			dll.head = nil
		} else if dll.tail != nil {
			dll.tail = nil
		}

		dll.size = 0
		return n
	}

	if n == dll.head {
		return dll.shift()
	}

	if n == dll.tail {
		return dll.pop()
	}

	n.next.prev = n.prev
	n.prev.next = n.next
	dll.size -= 1
	return n
}
