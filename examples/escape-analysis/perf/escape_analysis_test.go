package perf

import (
	"fmt"
	"testing"
)

type Data struct {
	A int
	B int
	C int
}

func StackAlloc() Data {
	return Data{1, 2, 3} // stays on stack
}

func HeapAlloc() *Data {
	return &Data{1, 2, 3} // escapes to heap
}

func BenchmarkStackAlloc(b *testing.B) {
	for b.Loop() {
		_ = StackAlloc()
	}
}

func BenchmarkHeapAlloc(b *testing.B) {
	c := 1
	for b.Loop() {
		if a := HeapAlloc(); a != nil {
			c = a.A
		}
	}
	fmt.Println(c) // prevent compiler optimization
}
