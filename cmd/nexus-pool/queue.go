package main

import (
	"sync/atomic"
)

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

func (q *taskQueue) push(task *Task) bool {
	for {
		tail := q.tail.Load()
		slot := &q.buffer[tail&q.mask]
		seq := slot.seq.Load()
		diff := int64(seq) - int64(tail)

		switch {
		case diff == 0:
			if q.tail.CompareAndSwap(tail, tail+1) {
				slot.task.Store(task)
				slot.seq.Store(tail + 1)
				return true
			}
		case diff < 0:
			return false
		default:
			continue
		}
	}
}

func (q *taskQueue) pop() (*Task, bool) {
	for {
		head := q.head.Load()
		slot := &q.buffer[head&q.mask]
		seq := slot.seq.Load()
		diff := int64(seq) - int64(head+1)

		switch {
		case diff == 0:
			if q.head.CompareAndSwap(head, head+1) {
				task := slot.task.Swap(nil)
				slot.seq.Store(head + q.mask + 1)
				return task, task != nil
			}
		case diff < 0:
			return nil, false
		default:
			continue
		}
	}
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
