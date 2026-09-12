package main

import (
	"os"
	"strings"
	"sync"
)

func logLine(line string) {
	f, _ := os.Create("output.txt")
	defer f.Close()

	f.WriteString(line + "\n")
}

var batch []string

func logBatch(line string) {
	f, _ := os.Create("output.txt")
	defer f.Close()

	batch = append(batch, line)
	if len(batch) >= 100 {
		f.WriteString(strings.Join(batch, "\n") + "\n")
		batch = batch[:0]
	}
}

type Batcher[T any] struct {
	mu     sync.Mutex
	buffer []T
	size   int
	flush  func([]T)
}

func NewBatcher[T any](size int, flush func([]T)) *Batcher[T] {
	return &Batcher[T]{
		buffer: make([]T, 0, size),
		size:   size,
		flush:  flush,
	}
}

func (b *Batcher[T]) Add(item T) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buffer = append(b.buffer, item)
	if len(b.buffer) >= b.size {
		b.flushNow()
	}
}

func (b *Batcher[T]) flushNow() {
	if len(b.buffer) == 0 {
		return
	}
	b.flush(b.buffer)
	b.buffer = b.buffer[:0]
}
