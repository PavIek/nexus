package ringbuffer

import "testing"

var okSink bool
var valSink int

func BenchmarkRingBufferPushPop(b *testing.B) {
	rb := NewRingBuffer[int](1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		okSink = rb.Push(i)
		valSink, okSink = rb.Pop()
	}
}
