package ringbuffer

import "sync/atomic"

type RingBuffer[T any] struct {
	buffer   []T
	writeIdx uint64
	readIdx  uint64
	mask     uint64
}

func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buffer: make([]T, capacity),
		mask:   uint64(capacity - 1),
	}
}

func (rb *RingBuffer[T]) Push(val T) bool {
	for {
		w := atomic.LoadUint64(&rb.writeIdx)
		r := atomic.LoadUint64(&rb.readIdx)
		if w-r >= uint64(len(rb.buffer)) {
			return false
		}
		if atomic.CompareAndSwapUint64(&rb.writeIdx, w, w+1) {
			rb.buffer[w&rb.mask] = val
			return true
		}
	}
}

func (rb *RingBuffer[T]) Pop() (T, bool) {
	var zero T
	for {
		r := atomic.LoadUint64(&rb.readIdx)
		w := atomic.LoadUint64(&rb.writeIdx)
		if r >= w {
			return zero, false
		}
		if atomic.CompareAndSwapUint64(&rb.readIdx, r, r+1) {
			return rb.buffer[r&rb.mask], true
		}
	}
}
