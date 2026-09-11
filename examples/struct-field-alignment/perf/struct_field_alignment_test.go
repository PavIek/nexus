package perf

import (
	"sync"
	"testing"
)

type PoorlyAligned struct {
	flag  bool
	count int64
	id    byte
}

type WellAligned struct {
	count int64
	flag  bool
	id    byte
}

func BenchmarkPoorlyAligned(b *testing.B) {
	for b.Loop() {
		var items = make([]PoorlyAligned, 10_000_000)
		for j := range items {
			items[j].count = int64(j)
		}
	}
}

func BenchmarkWellAligned(b *testing.B) {
	for b.Loop() {
		var items = make([]WellAligned, 10_000_000)
		for j := range items {
			items[j].count = int64(j)
		}
	}
}

type SharedCounterBad struct {
	a int64
	b int64
}

type SharedCounterGood struct {
	a int64
	_ [56]byte
	b int64
}

func BenchmarkFalseSharing(b *testing.B) {
	var c SharedCounterBad
	var wg sync.WaitGroup

	for b.Loop() {
		wg.Add(2)
		go func() {
			for i := 0; i < 1_000_000; i++ {
				c.a++
			}
			wg.Done()
		}()
		go func() {
			for i := 0; i < 1_000_000; i++ {
				c.b++
			}
			wg.Done()
		}()
		wg.Wait()
	}
}

func BenchmarkNoFalseSharing(b *testing.B) {
	var c SharedCounterGood
	var wg sync.WaitGroup

	for b.Loop() {
		wg.Add(2)
		go func() {
			for i := 0; i < 1_000_000; i++ {
				c.a++
			}
			wg.Done()
		}()
		go func() {
			for i := 0; i < 1_000_000; i++ {
				c.b++
			}
			wg.Done()
		}()
		wg.Wait()
	}
}
