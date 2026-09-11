package main

import "fmt"

func main() {
	resizeSlice()
}

func resizeSlice() {
	s := make([]int, 0)
	for i := 0; i < 10_000; i++ {
		s = append(s, i)
		fmt.Printf("Len: %d, Cap: %d\n", len(s), cap(s))
	}
}

func preallocationSlice() {
	result := make([]int, 0, 10_000)
	for i := 0; i < 10_000; i++ {
		result = append(result, i)
	}
}

func preallocationSliceWithoutBoundCheck() {
	result := make([]int, 10_000)
	for i := 0; i < 10_000; i++ {
		result[i] = i
	}
}
