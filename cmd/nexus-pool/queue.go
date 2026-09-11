package main

import "sync/atomic"

type taskQueue struct {
	buffer []queueSlot
	mask   uint64
	head   atomic.Uint64
	tail   atomic.Uint64
}

type queueSlot struct {
	seq  atomic.Uint64
	task atomic.Pointer[Task]
}

func newTaskQueue(capacity int) *taskQueue {
	capacity = nextPowerOfTwo(capacity)
	q := &taskQueue{
		buffer: make([]queueSlot, capacity),
		mask:   uint64(capacity - 1),
	}
	for i := range q.buffer {
		q.buffer[i].seq.Store(uint64(i))
	}
	return q
}

func nextPowerOfTwo(n int) int {
	if n < 2 {
		return 2
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}
